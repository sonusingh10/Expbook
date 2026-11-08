package models

import (
	"github.com/mayswind/ezbookkeeping/pkg/investments"
)

// Price sources for a security (Expbook)
const (
	SecurityPriceSourceManual = "manual"
	SecurityPriceSourceYahoo  = "yahoo"
	SecurityPriceSourceAmfi   = "amfi"
)

// Security represents something that can be held in a portfolio, stored in database (Expbook)
type Security struct {
	SecurityId      int64  `xorm:"PK"`
	Uid             int64  `xorm:"INDEX(IDX_security_uid_deleted) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_security_uid_deleted) NOT NULL"`
	Name            string `xorm:"VARCHAR(64) NOT NULL"`
	Symbol          string `xorm:"VARCHAR(32) NOT NULL"`
	SecurityType    int32  `xorm:"NOT NULL"`
	Currency        string `xorm:"VARCHAR(3) NOT NULL"`
	PriceSource     string `xorm:"VARCHAR(16) NOT NULL"`
	PriceSymbol     string `xorm:"VARCHAR(64) NOT NULL"`
	LatestPrice     int64  `xorm:"NOT NULL"`
	LatestPriceTime int64  `xorm:"NOT NULL"`
	Comment         string `xorm:"VARCHAR(255) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// SecurityPrice represents one stored price observation of a security (Expbook)
type SecurityPrice struct {
	PriceId    int64  `xorm:"PK"`
	Uid        int64  `xorm:"INDEX(IDX_security_price_uid_security_time) NOT NULL"`
	SecurityId int64  `xorm:"INDEX(IDX_security_price_uid_security_time) NOT NULL"`
	PriceTime  int64  `xorm:"INDEX(IDX_security_price_uid_security_time) NOT NULL"`
	Price      int64  `xorm:"NOT NULL"`
	Source     string `xorm:"VARCHAR(16) NOT NULL"`
}

// SecurityCreateRequest represents all parameters of security creation request
type SecurityCreateRequest struct {
	Name         string `json:"name" binding:"required,notBlank,max=64"`
	Symbol       string `json:"symbol" binding:"required,notBlank,max=32"`
	SecurityType int32  `json:"securityType" binding:"required,min=1,max=5"`
	Currency     string `json:"currency" binding:"required,len=3,validCurrency"`
	PriceSource  string `json:"priceSource" binding:"omitempty,max=16"`
	PriceSymbol  string `json:"priceSymbol" binding:"max=64"`
	Comment      string `json:"comment" binding:"max=255"`
}

// SecurityModifyRequest represents all parameters of security modification request
type SecurityModifyRequest struct {
	Id           int64  `json:"id,string" binding:"required,min=1"`
	Name         string `json:"name" binding:"required,notBlank,max=64"`
	Symbol       string `json:"symbol" binding:"required,notBlank,max=32"`
	SecurityType int32  `json:"securityType" binding:"required,min=1,max=5"`
	Currency     string `json:"currency" binding:"required,len=3,validCurrency"`
	PriceSource  string `json:"priceSource" binding:"omitempty,max=16"`
	PriceSymbol  string `json:"priceSymbol" binding:"max=64"`
	Comment      string `json:"comment" binding:"max=255"`
}

// SecurityDeleteRequest represents all parameters of security deleting request
type SecurityDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// SecuritySetPriceRequest represents all parameters of manually setting a security price
type SecuritySetPriceRequest struct {
	Id    int64  `json:"id,string" binding:"required,min=1"`
	Price string `json:"price" binding:"required"`
	Time  int64  `json:"time" binding:"min=0"`
}

// SecurityInfoResponse represents a view-object of security
type SecurityInfoResponse struct {
	Id              int64  `json:"id,string"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	SecurityType    int32  `json:"securityType"`
	Currency        string `json:"currency"`
	PriceSource     string `json:"priceSource"`
	PriceSymbol     string `json:"priceSymbol"`
	LatestPrice     string `json:"latestPrice"`
	LatestPriceTime int64  `json:"latestPriceTime"`
	Comment         string `json:"comment"`
}

// ToSecurityInfoResponse returns a view-object according to database model
func (s *Security) ToSecurityInfoResponse() *SecurityInfoResponse {
	latestPrice := ""

	if s.LatestPrice > 0 {
		latestPrice = investments.FormatScaled(s.LatestPrice)
	}

	return &SecurityInfoResponse{
		Id:              s.SecurityId,
		Name:            s.Name,
		Symbol:          s.Symbol,
		SecurityType:    s.SecurityType,
		Currency:        s.Currency,
		PriceSource:     s.PriceSource,
		PriceSymbol:     s.PriceSymbol,
		LatestPrice:     latestPrice,
		LatestPriceTime: s.LatestPriceTime,
		Comment:         s.Comment,
	}
}

// IsValidSecurityPriceSource returns whether the price source is known
func IsValidSecurityPriceSource(source string) bool {
	switch source {
	case SecurityPriceSourceManual, SecurityPriceSourceYahoo, SecurityPriceSourceAmfi:
		return true
	}

	return false
}

// SecurityPriceRefreshResult is the outcome of refreshing one security's price (Expbook)
type SecurityPriceRefreshResult struct {
	SecurityId int64  `json:"securityId,string"`
	Symbol     string `json:"symbol"`
	Ok         bool   `json:"ok"`
	Price      string `json:"price"`
	PriceTime  int64  `json:"priceTime"`
	Message    string `json:"message"`
}

// SecurityPriceRefreshResponse is the response of a price refresh
type SecurityPriceRefreshResponse struct {
	Results []*SecurityPriceRefreshResult `json:"results"`
}
