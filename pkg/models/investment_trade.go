package models

import (
	"github.com/mayswind/ezbookkeeping/pkg/investments"
)

// Trade sources (Expbook)
const (
	InvestmentTradeSourceManual = "manual"
	InvestmentTradeSourceCsv    = "csv"
	InvestmentTradeSourceBroker = "broker"
)

// InvestmentTrade represents one dated buy, sell or dividend on a security, stored in database (Expbook)
type InvestmentTrade struct {
	TradeId         int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_investment_trade_uid_deleted_security) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_investment_trade_uid_deleted_security) NOT NULL"`
	SecurityId      int64  `xorm:"INDEX(IDX_investment_trade_uid_deleted_security) NOT NULL"`
	TradeType       int32  `xorm:"NOT NULL"`
	TradeTime       int64  `xorm:"NOT NULL"`
	Quantity        int64  `xorm:"NOT NULL"`
	Price           int64  `xorm:"NOT NULL"`
	Amount          int64  `xorm:"NOT NULL"`
	Fees            int64  `xorm:"NOT NULL"`
	AccountId       int64  `xorm:"NOT NULL DEFAULT 0"`
	Source          string `xorm:"INDEX(IDX_investment_trade_uid_source_external) VARCHAR(16) NOT NULL"`
	ExternalId      string `xorm:"INDEX(IDX_investment_trade_uid_source_external) VARCHAR(64) NOT NULL"`
	Comment         string `xorm:"VARCHAR(255) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// ToLedgerTrade converts the database model to the pure ledger type
func (t *InvestmentTrade) ToLedgerTrade() investments.Trade {
	return investments.Trade{
		ID:       t.TradeId,
		Time:     t.TradeTime,
		Type:     investments.TradeType(t.TradeType),
		Quantity: t.Quantity,
		Price:    t.Price,
		Amount:   t.Amount,
		Fees:     t.Fees,
	}
}

// InvestmentTradeCreateRequest represents all parameters of trade creation request
type InvestmentTradeCreateRequest struct {
	SecurityId int64  `json:"securityId,string" binding:"required,min=1"`
	TradeType  int32  `json:"tradeType" binding:"required,min=1,max=3"`
	Time       int64  `json:"time" binding:"required,min=1"`
	Quantity   string `json:"quantity"`
	Price      string `json:"price"`
	Amount     int64  `json:"amount" binding:"min=0"`
	Fees       int64  `json:"fees" binding:"min=0"`
	AccountId  int64  `json:"accountId,string" binding:"min=0"`
	Comment    string `json:"comment" binding:"max=255"`
}

// InvestmentTradeModifyRequest represents all parameters of trade modification request
type InvestmentTradeModifyRequest struct {
	Id         int64  `json:"id,string" binding:"required,min=1"`
	SecurityId int64  `json:"securityId,string" binding:"required,min=1"`
	TradeType  int32  `json:"tradeType" binding:"required,min=1,max=3"`
	Time       int64  `json:"time" binding:"required,min=1"`
	Quantity   string `json:"quantity"`
	Price      string `json:"price"`
	Amount     int64  `json:"amount" binding:"min=0"`
	Fees       int64  `json:"fees" binding:"min=0"`
	AccountId  int64  `json:"accountId,string" binding:"min=0"`
	Comment    string `json:"comment" binding:"max=255"`
}

// InvestmentTradeDeleteRequest represents all parameters of trade deleting request
type InvestmentTradeDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// InvestmentTradeListRequest represents all parameters of trade list request
type InvestmentTradeListRequest struct {
	SecurityId int64 `form:"securityId,string" binding:"min=0"`
}

// InvestmentTradeInfoResponse represents a view-object of trade
type InvestmentTradeInfoResponse struct {
	Id         int64  `json:"id,string"`
	SecurityId int64  `json:"securityId,string"`
	TradeType  int32  `json:"tradeType"`
	Time       int64  `json:"time"`
	Quantity   string `json:"quantity"`
	Price      string `json:"price"`
	Amount     int64  `json:"amount"`
	Fees       int64  `json:"fees"`
	AccountId  int64  `json:"accountId,string"`
	Source     string `json:"source"`
	ExternalId string `json:"externalId"`
	Comment    string `json:"comment"`
}

// ToInvestmentTradeInfoResponse returns a view-object according to database model
func (t *InvestmentTrade) ToInvestmentTradeInfoResponse() *InvestmentTradeInfoResponse {
	return &InvestmentTradeInfoResponse{
		Id:         t.TradeId,
		SecurityId: t.SecurityId,
		TradeType:  t.TradeType,
		Time:       t.TradeTime,
		Quantity:   investments.FormatScaled(t.Quantity),
		Price:      investments.FormatScaled(t.Price),
		Amount:     t.Amount,
		Fees:       t.Fees,
		AccountId:  t.AccountId,
		Source:     t.Source,
		ExternalId: t.ExternalId,
		Comment:    t.Comment,
	}
}

// InvestmentTradeImportRequest represents all parameters of CSV import request
type InvestmentTradeImportRequest struct {
	Csv    string `json:"csv" binding:"required"`
	DryRun bool   `json:"dryRun"`
}

// InvestmentTradeSyncItem is one externally sourced trade (for broker sync)
type InvestmentTradeSyncItem struct {
	ExternalId   string `json:"externalId" binding:"required,max=64"`
	Symbol       string `json:"symbol" binding:"required,max=32"`
	Name         string `json:"name" binding:"max=64"`
	SecurityType int32  `json:"securityType" binding:"min=0,max=5"`
	Currency     string `json:"currency" binding:"required,len=3"`
	TradeType    int32  `json:"tradeType" binding:"required,min=1,max=3"`
	Time         int64  `json:"time" binding:"required,min=1"`
	Quantity     string `json:"quantity"`
	Price        string `json:"price"`
	Amount       int64  `json:"amount" binding:"min=0"`
	Fees         int64  `json:"fees" binding:"min=0"`
	Comment      string `json:"comment" binding:"max=255"`
}

// InvestmentTradeSyncRequest represents all parameters of external trade sync request
type InvestmentTradeSyncRequest struct {
	Items  []*InvestmentTradeSyncItem `json:"items" binding:"required,min=1,max=5000"`
	DryRun bool                       `json:"dryRun"`
}

// Import row statuses
const (
	InvestmentImportRowValid     = "valid"
	InvestmentImportRowDuplicate = "duplicate"
	InvestmentImportRowError     = "error"
)

// InvestmentImportRowResponse is the outcome of one import row
type InvestmentImportRowResponse struct {
	Line    int    `json:"line"`
	Symbol  string `json:"symbol"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// InvestmentImportResponse is the result of an import preview or run
type InvestmentImportResponse struct {
	DryRun            bool                           `json:"dryRun"`
	Rows              []*InvestmentImportRowResponse `json:"rows"`
	Created           int                            `json:"created"`
	Skipped           int                            `json:"skipped"`
	Failed            int                            `json:"failed"`
	CreatedSecurities int                            `json:"createdSecurities"`
}
