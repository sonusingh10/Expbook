import json
import os
import unittest
from datetime import date, datetime, timezone

import fyers_to_expbook as conv

HERE = os.path.dirname(os.path.abspath(__file__))


def fixture(name):
    return os.path.join(HERE, "fixtures", name)


class TradeConversion(unittest.TestCase):
    def setUp(self):
        records = conv.load_records(fixture("fyers_tradebook_sample.json"), ("tradeBook",))
        self.items, self.skipped, self.problems = conv.convert_trades(records)

    def test_equity_trades_become_sync_items(self):
        self.assertEqual(len(self.items), 4)
        buy = self.items[0]
        self.assertEqual(buy["externalId"], "fyers:DEMO-T-1001")
        self.assertEqual(buy["symbol"], "DEMOINFY")
        self.assertEqual((buy["tradeType"], buy["quantity"], buy["price"]), (1, "10", "1400.5"))
        self.assertEqual(self.items[2]["tradeType"], 2)

    def test_dates_are_calendar_days_stored_at_noon_utc(self):
        expected = int(datetime(2025, 10, 4, 12, tzinfo=timezone.utc).timestamp())
        self.assertEqual(self.items[0]["time"], expected)

    def test_etf_is_detected(self):
        self.assertEqual(self.items[3]["securityType"], conv.SECURITY_TYPE_ETF)
        self.assertEqual(self.items[0]["securityType"], conv.SECURITY_TYPE_STOCK)

    def test_derivatives_and_duplicates_are_skipped_not_guessed(self):
        reasons = {symbol: reason for _, symbol, reason in self.skipped}
        self.assertIn("segment 11", reasons["NSE:NIFTY26NOVFUT"])
        self.assertIn("duplicate trade id", reasons["NSE:DEMOINFY-EQ"])

    def test_bad_rows_are_reported_with_their_position(self):
        self.assertEqual(len(self.problems), 1)
        self.assertEqual(self.problems[0][0], 7)
        self.assertIn("unknown side", self.problems[0][1])

    def test_ids_are_stable_between_runs(self):
        again, _, _ = conv.convert_trades(conv.load_records(fixture("fyers_tradebook_sample.json"), ("tradeBook",)))
        self.assertEqual([i["externalId"] for i in again], [i["externalId"] for i in self.items])

    def test_no_trade_number_falls_back_to_order_number_quantity_price(self):
        items, _, problems = conv.convert_trades([
            {"symbol": "NSE:DEMOX-EQ", "side": "BUY", "qty": "2", "price": "10.5", "time": "2026-01-02 10:00:00", "orderNumber": "O9"}
        ])
        self.assertEqual(problems, [])
        self.assertEqual(items[0]["externalId"], "fyers:O9-2-10.5")


class HoldingsConversion(unittest.TestCase):
    def test_opening_balances_use_cost_price_and_skip_zero_quantity(self):
        records = conv.load_records(fixture("fyers_holdings_sample.json"), ("holdings",))
        items, skipped, problems = conv.convert_holdings(records, date(2025, 4, 1))
        self.assertEqual(problems, [])
        self.assertEqual(len(items), 1)
        self.assertEqual(items[0]["externalId"], "fyers-opening:DEMOTCS:2025-04-01")
        self.assertEqual((items[0]["quantity"], items[0]["price"]), ("8", "3200.25"))
        self.assertEqual(len(skipped), 1)


class Parsing(unittest.TestCase):
    def test_symbol_split(self):
        self.assertEqual(conv.split_symbol("NSE:SBIN-EQ"), ("NSE", "SBIN"))
        self.assertEqual(conv.split_symbol("BSE:500325-EQ"), ("BSE", "500325"))
        self.assertIsNone(conv.split_symbol("NSE:NIFTY26NOVFUT"))
        self.assertIsNone(conv.split_symbol("NSE:NIFTY2611021500CE"))

    def test_unknown_file_shape_names_the_keys_found(self):
        path = os.path.join(HERE, "fixtures", "_odd.json")
        with open(path, "w") as handle:
            json.dump({"weird": 1}, handle)
        try:
            with self.assertRaises(conv.ConversionError) as ctx:
                conv.load_records(path, ("tradeBook",))
            self.assertIn("weird", str(ctx.exception))
        finally:
            os.remove(path)

    def test_decimals_are_plain_and_capped_at_eight_places(self):
        self.assertEqual(conv.to_decimal_text(10), "10")
        self.assertEqual(conv.to_decimal_text("0.123456789"), "0.12345678")
        with self.assertRaises(conv.ConversionError):
            conv.to_decimal_text("-1")


if __name__ == "__main__":
    unittest.main()
