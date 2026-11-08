#!/usr/bin/env python3
"""Convert a Fyers trade book (and optionally holdings) into an Expbook trade-sync payload.

Reads JSON that you saved from the broker (for example the output of the Fyers
`get_trade_history` and `get_holdings` tools, or the Fyers API v3 tradebook / holdings responses),
writes the payload for `POST /api/v1/investments/trades/sync.json`, and can post it for you.

Security: the Expbook API token is read from the EXPBOOK_TOKEN environment variable and the server URL from
EXPBOOK_URL (default http://127.0.0.1:8080). Nothing is read from or written to the repository, and this
script never talks to the broker. Standard library only.

Idempotent: every trade gets a stable external id (`fyers:<trade number>`), so running the same file again
creates nothing new on the server.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request
from decimal import Decimal
from datetime import date, datetime, timedelta, timezone

SECURITY_TYPE_STOCK = 1
SECURITY_TYPE_ETF = 2
SECURITY_TYPE_MUTUAL_FUND = 3

TRADE_BUY = 1
TRADE_SELL = 2

# Fyers segment codes that map to cash-market equity. Derivatives, currency and commodities are skipped.
EQUITY_SEGMENTS = {10, "10", "Equity", "equity", "CM", "cm"}
EQUITY_SYMBOL = re.compile(r"^(?P<exchange>[A-Z]+):(?P<name>[A-Z0-9&_.\-]+?)(?:-(?P<series>EQ|BE|BZ|SM|ST|N1|IL))?$")

# Field name aliases, so the script tolerates small differences between the API and the MCP tool output.
ALIASES = {
    "symbol": ("symbol", "tradingSymbol", "trading_symbol"),
    "trade_id": ("tradeNumber", "trade_number", "tradeId", "trade_id", "exchangeOrderNo"),
    "order_id": ("orderNumber", "order_number", "orderId", "order_id"),
    "side": ("side", "transactionType", "transaction_type"),
    "quantity": ("tradedQty", "traded_qty", "quantity", "qty"),
    "price": ("tradePrice", "traded_price", "price"),
    "time": ("orderDateTime", "order_date_time", "tradeTime", "trade_time", "time"),
    "segment": ("segment",),
}

HOLDING_ALIASES = {
    "symbol": ("symbol", "tradingSymbol", "trading_symbol"),
    "quantity": ("quantity", "remainingQuantity", "qty"),
    "cost": ("costPrice", "cost_price", "avgPrice", "average_price"),
}

TIME_FORMATS = ("%d-%b-%Y %H:%M:%S", "%Y-%m-%d %H:%M:%S", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%d", "%d-%m-%Y %H:%M:%S")


INDIA = timezone(timedelta(hours=5, minutes=30))


class ConversionError(Exception):
    pass


def pick(record: dict, names: tuple):
    for name in names:
        if name in record and record[name] not in (None, ""):
            return record[name]
    return None


def load_records(path: str, keys: tuple) -> list:
    with open(path, "r", encoding="utf-8") as handle:
        data = json.load(handle)

    if isinstance(data, list):
        return data

    if isinstance(data, dict):
        for key in keys:
            value = data.get(key)

            if isinstance(value, list):
                return value

            if isinstance(value, dict):
                for nested in keys:
                    if isinstance(value.get(nested), list):
                        return value[nested]

    raise ConversionError(
        f"{path}: expected a JSON list or an object with one of {list(keys)}; "
        f"found top-level keys {sorted(data) if isinstance(data, dict) else type(data).__name__}"
    )


def split_symbol(symbol: str):
    """NSE:SBIN-EQ -> ("NSE", "SBIN"); returns None for anything that is not a plain cash-market symbol."""
    match = EQUITY_SYMBOL.match(symbol.strip().upper())

    if not match:
        return None

    # Derivative symbols such as NSE:NIFTY24NOVFUT or NSE:NIFTY2411421500CE carry no equity series.
    if match.group("series") is None and re.search(r"(FUT|CE|PE)$", match.group("name")):
        return None

    return match.group("exchange"), match.group("name")


def security_type_for(name: str) -> int:
    upper = name.upper()

    if upper.endswith("BEES") or "ETF" in upper:
        return SECURITY_TYPE_ETF

    return SECURITY_TYPE_STOCK


def parse_calendar_date(value) -> date:
    """The calendar day in India (the broker's local day); the time of day is dropped."""
    if isinstance(value, (int, float)):
        return datetime.fromtimestamp(float(value), tz=timezone.utc).astimezone(INDIA).date()

    text = str(value).strip()

    for fmt in TIME_FORMATS:
        try:
            return datetime.strptime(text, fmt).date()
        except ValueError:
            continue

    raise ConversionError(f"unrecognised date/time '{text}'")


def noon_utc(day: date) -> int:
    """Trades are calendar dates, stored at 12:00 UTC like the Expbook screens do."""
    return int(datetime(day.year, day.month, day.day, 12, tzinfo=timezone.utc).timestamp())


def to_decimal_text(value) -> str:
    text = format(Decimal(repr(value)), "f") if isinstance(value, float) else str(value).strip()

    if not re.fullmatch(r"\d+(\.\d+)?", text):
        raise ConversionError(f"'{value}' is not a plain positive decimal number")

    whole, _, fraction = text.partition(".")

    fraction = fraction[:8].rstrip("0")

    return whole + ("." + fraction if fraction else "")


def convert_trades(records: list) -> tuple:
    items, skipped, problems = [], [], []
    seen = set()

    for index, record in enumerate(records, start=1):
        try:
            symbol_text = pick(record, ALIASES["symbol"])

            if not symbol_text:
                raise ConversionError("no symbol")

            segment = pick(record, ALIASES["segment"])

            if segment is not None and segment not in EQUITY_SEGMENTS:
                skipped.append((index, symbol_text, f"segment {segment} is not cash-market equity"))
                continue

            parts = split_symbol(str(symbol_text))

            if parts is None:
                skipped.append((index, symbol_text, "not a cash-market equity symbol"))
                continue

            _, name = parts
            side = pick(record, ALIASES["side"])
            side_text = str(side).strip().upper()
            trade_type = TRADE_BUY if side_text in ("1", "BUY", "B") else TRADE_SELL if side_text in ("-1", "SELL", "S") else None

            if trade_type is None:
                raise ConversionError(f"unknown side '{side}'")

            quantity = to_decimal_text(pick(record, ALIASES["quantity"]))
            price = to_decimal_text(pick(record, ALIASES["price"]))
            when = parse_calendar_date(pick(record, ALIASES["time"]))

            trade_id = pick(record, ALIASES["trade_id"])

            if trade_id is None:
                order_id = pick(record, ALIASES["order_id"])

                if order_id is None:
                    raise ConversionError("no trade number or order number to build a stable id from")

                trade_id = f"{order_id}-{quantity}-{price}"

            external_id = f"fyers:{trade_id}"[:64]

            if external_id in seen:
                skipped.append((index, symbol_text, "duplicate trade id in the file"))
                continue

            seen.add(external_id)
            items.append({
                "externalId": external_id,
                "symbol": name,
                "name": name,
                "securityType": security_type_for(name),
                "currency": "INR",
                "tradeType": trade_type,
                "time": noon_utc(when),
                "quantity": quantity,
                "price": price,
                "amount": 0,
                "fees": 0,
                "comment": "Fyers",
            })
        except ConversionError as error:
            problems.append((index, str(error)))

    return items, skipped, problems


def convert_holdings(records: list, opening_date: date) -> tuple:
    """Opening balances for holdings whose buys are older than the trade history you have."""
    items, skipped, problems = [], [], []

    for index, record in enumerate(records, start=1):
        try:
            symbol_text = pick(record, HOLDING_ALIASES["symbol"])
            parts = split_symbol(str(symbol_text or ""))

            if parts is None:
                skipped.append((index, symbol_text, "not a cash-market equity symbol"))
                continue

            _, name = parts
            quantity = pick(record, HOLDING_ALIASES["quantity"])
            cost = pick(record, HOLDING_ALIASES["cost"])

            if quantity is None or cost is None:
                raise ConversionError("holding has no quantity or cost price")

            if float(quantity) <= 0:
                skipped.append((index, symbol_text, "zero quantity"))
                continue

            items.append({
                "externalId": f"fyers-opening:{name}:{opening_date.isoformat()}"[:64],
                "symbol": name,
                "name": name,
                "securityType": security_type_for(name),
                "currency": "INR",
                "tradeType": TRADE_BUY,
                "time": noon_utc(opening_date),
                "quantity": to_decimal_text(quantity),
                "price": to_decimal_text(cost),
                "amount": 0,
                "fees": 0,
                "comment": f"Fyers opening balance on {opening_date.isoformat()} (average cost)",
            })
        except ConversionError as error:
            problems.append((index, str(error)))

    return items, skipped, problems


def post_json(url: str, token: str, body: dict) -> dict:
    request = urllib.request.Request(
        url,
        data=json.dumps(body).encode("utf-8"),
        method="POST",
        headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
    )

    try:
        with urllib.request.urlopen(request, timeout=120) as response:
            return json.loads(response.read())
    except urllib.error.HTTPError as error:
        return json.loads(error.read() or b"{}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--trades", help="JSON file with the Fyers trade book")
    parser.add_argument("--holdings", help="JSON file with Fyers holdings, used for opening balances")
    parser.add_argument("--opening-date", help="YYYY-MM-DD: the day opening balances are dated (required with --holdings)")
    parser.add_argument("--out", help="write the sync payload here (default: stdout when not posting)")
    parser.add_argument("--post", action="store_true", help="post the payload to Expbook (needs EXPBOOK_TOKEN)")
    parser.add_argument("--dry-run", action="store_true", help="with --post: ask the server to validate only, saving nothing")
    return parser


def report(label: str, skipped: list, problems: list) -> None:
    for index, symbol, reason in skipped:
        print(f"  {label} #{index} {symbol}: skipped, {reason}", file=sys.stderr)

    for index, reason in problems:
        print(f"  {label} #{index}: error, {reason}", file=sys.stderr)


def main(argv=None) -> int:
    args = build_parser().parse_args(argv)

    if not args.trades and not args.holdings:
        print("Give --trades and/or --holdings.", file=sys.stderr)
        return 2

    items = []

    try:
        if args.holdings:
            if not args.opening_date:
                print("--holdings needs --opening-date.", file=sys.stderr)
                return 2

            opening = date.fromisoformat(args.opening_date)
            converted, skipped, problems = convert_holdings(load_records(args.holdings, ("holdings", "data", "result")), opening)
            report("holding", skipped, problems)
            items += converted

        if args.trades:
            converted, skipped, problems = convert_trades(load_records(args.trades, ("tradeBook", "trades", "data", "result")))
            report("trade", skipped, problems)
            items += converted
    except (ConversionError, OSError, ValueError, json.JSONDecodeError) as error:
        print(f"Error: {error}", file=sys.stderr)
        return 1

    if not items:
        print("Nothing to sync.", file=sys.stderr)
        return 1

    payload = {"items": items, "dryRun": bool(args.dry_run)}
    print(f"Prepared {len(items)} trades.", file=sys.stderr)

    if args.out:
        with open(args.out, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, indent=2)

    if not args.post:
        if not args.out:
            json.dump(payload, sys.stdout, indent=2)
            print()

        return 0

    token = os.environ.get("EXPBOOK_TOKEN", "")

    if not token:
        print("Set EXPBOOK_TOKEN to an Expbook API token before using --post.", file=sys.stderr)
        return 2

    base = os.environ.get("EXPBOOK_URL", "http://127.0.0.1:8080").rstrip("/")
    response = post_json(f"{base}/api/v1/investments/trades/sync.json", token, payload)

    if not response.get("success"):
        print(f"Server refused the request: {response.get('errorMessage', response)}", file=sys.stderr)
        return 1

    result = response["result"]
    verb = "Dry run, nothing saved. Would create" if args.dry_run else "Created"
    print(f"{verb} {result.get('created')}, skipped as duplicates {result.get('skipped')}, failed {result.get('failed')}, new securities {result.get('createdSecurities')}.")

    for row in result.get("rows", []):
        if row.get("status") == "error":
            print(f"  line {row.get('line')} {row.get('symbol')}: {row.get('message')}", file=sys.stderr)

    return 0 if not result.get("failed") else 1


if __name__ == "__main__":
    sys.exit(main())
