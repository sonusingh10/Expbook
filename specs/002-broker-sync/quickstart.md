# Quickstart: Fyers to Expbook

## 1. Get the broker data (you do this, in Cursor)

With the Fyers MCP server enabled, ask the agent to log in and save two files outside the repository, for example in `~/expbook-import/` (they contain your account data):

- the result of `get_trade_history` (use `from_date` as far back as Fyers allows, `segment_type` 1 for equity) as `tradebook.json`
- the result of `get_holdings` as `holdings.json` (only needed for opening balances)

The converter never contacts Fyers, and it needs no Fyers credentials.

## 2. Create an Expbook API token

API tokens are off by default. Turn them on for your server with `EBK_SECURITY_ENABLE_API_TOKEN=true` (in your untracked `.env`), restart, then sign in, open Settings, and create an API token under your sessions. Put it in your shell, not in a file in the repository:

```sh
export EXPBOOK_TOKEN='...'            # never commit this
export EXPBOOK_URL='http://127.0.0.1:8080'   # optional; this is the default
```

## 3. Preview, then sync

```sh
cd scripts/broker
# 1) preview: the server validates and saves nothing
python3 fyers_to_expbook.py --trades ~/expbook-import/tradebook.json --post --dry-run

# 2) real run
python3 fyers_to_expbook.py --trades ~/expbook-import/tradebook.json --post

# with opening balances for holdings older than your trade history
python3 fyers_to_expbook.py --trades ~/expbook-import/tradebook.json \
  --holdings ~/expbook-import/holdings.json --opening-date 2025-04-01 --post
```

Run it again any time: trades already synced are reported as duplicates and nothing is created. To inspect the payload without posting, use `--out payload.json` (no `--post`).

## 4. After the first sync

- Synced securities start with a manual price. In Investments choose Edit Security, set Price source to Yahoo and Price symbol to `SYMBOL.NS` (for example `SBIN.NS`), then Refresh prices. Or set the price by hand.
- Fees are 0 (the Fyers trade book has no charges). Edit a trade to add its fees if you want exact average cost.
- If a sell is refused with "sell quantity exceeds the quantity held", the buys are older than your history: add `--holdings` and `--opening-date` set to the day before your trade history starts.

## Tests

```sh
cd scripts/broker && python3 -m unittest -v test_fyers_to_expbook
```

The fixtures are synthetic (`DEMO...` symbols) and contain no real account data.
