package investments

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// MaxExternalIDLength is the longest external id that fits the database column
const MaxExternalIDLength = 64

// ParsedTrade is one validated CSV row, not yet tied to a stored security
type ParsedTrade struct {
	Line         int
	Symbol       string
	Name         string
	SecurityType SecurityType
	Currency     string
	Type         TradeType
	Time         int64
	Quantity     int64
	Price        int64
	Amount       int64
	Fees         int64
	ExternalID   string
	Account      string
	Comment      string
}

// RowResult is the outcome of parsing one data line
type RowResult struct {
	Line  int
	Trade *ParsedTrade
	Error string
}

// ErrCsvEmpty is returned when the file has no content
var ErrCsvEmpty = errors.New("csv content is empty")

// ErrCsvMissingColumns is returned when required header columns are absent
var ErrCsvMissingColumns = errors.New("csv header is missing required columns")

var headerAliases = map[string]string{
	"date": "date", "time": "date", "datetime": "date", "trade date": "date", "trade_date": "date",
	"type": "type", "trade type": "type", "trade_type": "type", "action": "type", "side": "type", "transaction type": "type",
	"symbol": "symbol", "ticker": "symbol", "code": "symbol", "isin": "symbol",
	"quantity": "quantity", "qty": "quantity", "units": "quantity", "shares": "quantity",
	"price": "price", "rate": "price", "nav": "price", "unit price": "price", "unit_price": "price",
	"amount": "amount", "value": "amount", "total": "amount",
	"fees": "fees", "fee": "fees", "brokerage": "fees", "charges": "fees", "commission": "fees",
	"currency": "currency", "ccy": "currency",
	"name": "name", "security": "name", "security name": "name", "security_name": "name",
	"security_type": "security_type", "security type": "security_type", "asset class": "security_type", "asset_type": "security_type",
	"external_id": "external_id", "external id": "external_id", "id": "external_id", "order id": "external_id", "trade id": "external_id",
	"account": "account",
	"note":    "note", "comment": "note", "notes": "note", "memo": "note",
}

// ParseCSV parses broker or spreadsheet CSV text into trades.
// The first row must be a header. Required columns: date, type, symbol.
// Buy and sell rows need quantity plus price (or amount, from which price is derived); dividends need amount.
// Rows are validated independently so one bad row never blocks the others.
func ParseCSV(content string, defaultCurrency string) ([]RowResult, error) {
	content = strings.TrimPrefix(content, "\ufeff")

	if strings.TrimSpace(content) == "" {
		return nil, ErrCsvEmpty
	}

	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()

	if err != nil {
		return nil, ErrCsvEmpty
	}

	columns := map[string]int{}

	for i, name := range header {
		key := strings.ToLower(strings.TrimSpace(name))

		if canonical, exists := headerAliases[key]; exists {
			if _, taken := columns[canonical]; !taken {
				columns[canonical] = i
			}
		}
	}

	var missing []string

	for _, required := range []string{"date", "type", "symbol"} {
		if _, ok := columns[required]; !ok {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrCsvMissingColumns, strings.Join(missing, ", "))
	}

	var results []RowResult
	line := 1

	for {
		record, err := reader.Read()
		line++

		if err == io.EOF {
			break
		}

		if err != nil {
			results = append(results, RowResult{Line: line, Error: "row could not be read: " + err.Error()})
			continue
		}

		if isBlankRecord(record) {
			continue
		}

		trade, message := parseRecord(record, columns, line, defaultCurrency)

		if message != "" {
			results = append(results, RowResult{Line: line, Error: message})
		} else {
			results = append(results, RowResult{Line: line, Trade: trade})
		}
	}

	return results, nil
}

func isBlankRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}

	return true
}

func field(record []string, columns map[string]int, name string) string {
	index, ok := columns[name]

	if !ok || index >= len(record) {
		return ""
	}

	return strings.TrimSpace(record[index])
}

func parseRecord(record []string, columns map[string]int, line int, defaultCurrency string) (*ParsedTrade, string) {
	trade := &ParsedTrade{Line: line}

	var err error

	trade.Time, err = parseTradeTime(field(record, columns, "date"))

	if err != nil {
		return nil, "date must be YYYY-MM-DD, YYYY-MM-DD HH:MM[:SS] or RFC3339"
	}

	trade.Type, err = parseTradeType(field(record, columns, "type"))

	if err != nil {
		return nil, "type must be buy, sell or dividend"
	}

	trade.Symbol = strings.ToUpper(field(record, columns, "symbol"))

	if trade.Symbol == "" || len(trade.Symbol) > 32 {
		return nil, "symbol is required and must be at most 32 characters"
	}

	trade.Name = field(record, columns, "name")

	if len(trade.Name) > 64 {
		return nil, "name must be at most 64 characters"
	}

	if trade.Name == "" {
		trade.Name = trade.Symbol
	}

	trade.Currency = strings.ToUpper(field(record, columns, "currency"))

	if trade.Currency == "" {
		trade.Currency = strings.ToUpper(defaultCurrency)
	}

	if len(trade.Currency) != 3 {
		return nil, "currency must be a 3-letter code"
	}

	trade.SecurityType = parseSecurityType(field(record, columns, "security_type"))
	trade.Account = field(record, columns, "account")
	trade.Comment = field(record, columns, "note")

	if len(trade.Comment) > 255 {
		return nil, "note must be at most 255 characters"
	}

	if raw := field(record, columns, "fees"); raw != "" {
		trade.Fees, err = ParseMinor(raw)

		if err != nil || trade.Fees < 0 {
			return nil, "fees must be a non-negative number with at most 2 decimals"
		}
	}

	switch trade.Type {
	case TradeTypeDividend:
		raw := field(record, columns, "amount")

		if raw == "" {
			return nil, "dividend rows need an amount"
		}

		trade.Amount, err = ParseMinor(raw)

		if err != nil || trade.Amount <= 0 {
			return nil, "amount must be a positive number with at most 2 decimals"
		}

	default:
		rawQuantity := field(record, columns, "quantity")

		if rawQuantity == "" {
			return nil, "quantity is required for buy and sell rows"
		}

		trade.Quantity, err = ParseScaled(rawQuantity)

		if err != nil || trade.Quantity <= 0 {
			return nil, "quantity must be a positive number with at most 8 decimals"
		}

		rawPrice := field(record, columns, "price")
		rawAmount := field(record, columns, "amount")

		switch {
		case rawPrice != "":
			trade.Price, err = ParseScaled(rawPrice)

			if err != nil || trade.Price <= 0 {
				return nil, "price must be a positive number with at most 8 decimals"
			}

		case rawAmount != "":
			amount, amountErr := ParseMinor(rawAmount)

			if amountErr != nil || amount <= 0 {
				return nil, "amount must be a positive number with at most 2 decimals"
			}

			trade.Price = PriceFromAmount(amount, trade.Quantity)

			if trade.Price <= 0 {
				return nil, "price could not be derived from amount and quantity"
			}

		default:
			return nil, "price (or amount) is required for buy and sell rows"
		}
	}

	trade.ExternalID = field(record, columns, "external_id")

	if len(trade.ExternalID) > MaxExternalIDLength {
		return nil, fmt.Sprintf("external id must be at most %d characters", MaxExternalIDLength)
	}

	if trade.ExternalID == "" {
		trade.ExternalID = TradeFingerprint(trade)
	}

	return trade, ""
}

// TradeFingerprint returns a stable idempotency key for a trade that has no external id.
// SHA-256 is used purely as a content fingerprint, not for any security purpose.
func TradeFingerprint(t *ParsedTrade) string {
	canonical := fmt.Sprintf("%d|%d|%s|%s|%d|%d|%d|%d", t.Time, t.Type, t.Symbol, t.Currency, t.Quantity, t.Price, t.Amount, t.Fees)
	sum := sha256.Sum256([]byte(canonical))

	return "h:" + hex.EncodeToString(sum[:16])
}

func parseTradeTime(value string) (int64, error) {
	value = strings.TrimSpace(value)

	if t, err := time.Parse("2006-01-02", value); err == nil {
		// date only: use midday UTC so the calendar day is the same in every timezone
		return t.Add(12 * time.Hour).Unix(), nil
	}

	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Unix(), nil
		}
	}

	return 0, errors.New("invalid date")
}

func parseTradeType(value string) (TradeType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "buy", "b", "purchase", "bought", "sip", "invest":
		return TradeTypeBuy, nil
	case "sell", "s", "sale", "sold", "redeem", "redemption":
		return TradeTypeSell, nil
	case "dividend", "div", "interest", "income", "payout":
		return TradeTypeDividend, nil
	}

	return 0, errors.New("invalid trade type")
}

func parseSecurityType(value string) SecurityType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "etf":
		return SecurityTypeETF
	case "fund", "mutual fund", "mutual_fund", "mf":
		return SecurityTypeMutualFund
	case "crypto", "cryptocurrency":
		return SecurityTypeCrypto
	case "other":
		return SecurityTypeOther
	default:
		return SecurityTypeStock
	}
}
