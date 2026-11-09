package api

import (
	"errors"
	"sort"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/exchangerates"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// maxInvestmentCsvBytes bounds the size of an imported CSV file
const maxInvestmentCsvBytes = 2 * 1024 * 1024

// InvestmentsApi represents the investments api (Expbook)
type InvestmentsApi struct {
	ApiUsingConfig
	users      *services.UserService
	accounts   *services.AccountService
	securities *services.SecurityService
	trades     *services.InvestmentTradeService
	prices     *services.InvestmentPriceService
}

// Initialize an investments api singleton instance
var (
	Investments = &InvestmentsApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		users:      services.Users,
		accounts:   services.Accounts,
		securities: services.Securities,
		trades:     services.InvestmentTrades,
		prices:     services.InvestmentPrices,
	}
)

// SecurityListHandler returns all securities of current user
func (a *InvestmentsApi) SecurityListHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	securities, err := a.securities.GetAllSecuritiesByUid(c, uid)

	if err != nil {
		log.Errorf(c, "[investments.SecurityListHandler] failed to get securities for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	resps := make([]*models.SecurityInfoResponse, len(securities))

	for i, security := range securities {
		resps[i] = security.ToSecurityInfoResponse()
	}

	sort.SliceStable(resps, func(i, j int) bool { return resps[i].Symbol < resps[j].Symbol })

	return resps, nil
}

// SecurityCreateHandler saves a new security for current user
func (a *InvestmentsApi) SecurityCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.SecurityCreateRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.SecurityCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	security := &models.Security{
		Uid:          uid,
		Name:         req.Name,
		Symbol:       req.Symbol,
		SecurityType: req.SecurityType,
		Currency:     req.Currency,
		PriceSource:  req.PriceSource,
		PriceSymbol:  req.PriceSymbol,
		Comment:      req.Comment,
	}

	err = a.securities.CreateSecurity(c, security)

	if err != nil {
		log.Errorf(c, "[investments.SecurityCreateHandler] failed to create security for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.SecurityCreateHandler] user \"uid:%d\" has created security \"id:%d\" successfully", uid, security.SecurityId)

	return security.ToSecurityInfoResponse(), nil
}

// SecurityModifyHandler saves an existing security for current user
func (a *InvestmentsApi) SecurityModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.SecurityModifyRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.SecurityModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	security := &models.Security{
		SecurityId:   req.Id,
		Uid:          uid,
		Name:         req.Name,
		Symbol:       req.Symbol,
		SecurityType: req.SecurityType,
		Currency:     req.Currency,
		PriceSource:  req.PriceSource,
		PriceSymbol:  req.PriceSymbol,
		Comment:      req.Comment,
	}

	err = a.securities.ModifySecurity(c, security)

	if err != nil {
		log.Errorf(c, "[investments.SecurityModifyHandler] failed to update security \"id:%d\" for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	saved, err := a.securities.GetSecurityBySecurityId(c, uid, req.Id)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.SecurityModifyHandler] user \"uid:%d\" has updated security \"id:%d\" successfully", uid, req.Id)

	return saved.ToSecurityInfoResponse(), nil
}

// SecurityDeleteHandler deletes a security that has no trades
func (a *InvestmentsApi) SecurityDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.SecurityDeleteRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.SecurityDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.securities.DeleteSecurity(c, uid, req.Id)

	if err != nil {
		log.Errorf(c, "[investments.SecurityDeleteHandler] failed to delete security \"id:%d\" for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.SecurityDeleteHandler] user \"uid:%d\" has deleted security \"id:%d\"", uid, req.Id)

	return true, nil
}

// SecuritySetPriceHandler stores a manual price for a security
func (a *InvestmentsApi) SecuritySetPriceHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.SecuritySetPriceRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.SecuritySetPriceHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	price, err := investments.ParseScaled(req.Price)

	if err != nil || price <= 0 {
		return nil, errs.ErrSecurityPriceInvalid
	}

	priceTime := req.Time

	if priceTime <= 0 {
		priceTime = time.Now().Unix()
	}

	uid := c.GetCurrentUid()
	err = a.securities.SetLatestPrice(c, uid, req.Id, price, priceTime, models.SecurityPriceSourceManual)

	if err != nil {
		log.Errorf(c, "[investments.SecuritySetPriceHandler] failed to set price of security \"id:%d\" for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	saved, err := a.securities.GetSecurityBySecurityId(c, uid, req.Id)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return saved.ToSecurityInfoResponse(), nil
}

// TradeListHandler returns trades of current user, newest first, optionally for one security
func (a *InvestmentsApi) TradeListHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentTradeListRequest
	err := c.ShouldBindQuery(&req)

	if err != nil {
		log.Warnf(c, "[investments.TradeListHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	var trades []*models.InvestmentTrade

	if req.SecurityId > 0 {
		trades, err = a.trades.GetTradesBySecurityId(c, uid, req.SecurityId)
	} else {
		trades, err = a.trades.GetAllTradesByUid(c, uid)
	}

	if err != nil {
		log.Errorf(c, "[investments.TradeListHandler] failed to get trades for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	sort.SliceStable(trades, func(i, j int) bool {
		if trades[i].TradeTime != trades[j].TradeTime {
			return trades[i].TradeTime > trades[j].TradeTime
		}

		return trades[i].TradeId > trades[j].TradeId
	})

	resps := make([]*models.InvestmentTradeInfoResponse, len(trades))

	for i, trade := range trades {
		resps[i] = trade.ToInvestmentTradeInfoResponse()
	}

	return resps, nil
}

func parseOptionalScaled(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}

	return investments.ParseScaled(value)
}

func (a *InvestmentsApi) checkAccount(c *core.WebContext, uid int64, accountId int64) error {
	if accountId <= 0 {
		return nil
	}

	_, err := a.accounts.GetAccountByAccountId(c, uid, accountId)

	return err
}

// TradeCreateHandler saves a new trade for current user
func (a *InvestmentsApi) TradeCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentTradeCreateRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.TradeCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	quantity, qErr := parseOptionalScaled(req.Quantity)
	price, pErr := parseOptionalScaled(req.Price)

	if qErr != nil || pErr != nil {
		return nil, errs.ErrInvestmentTradeValueInvalid
	}

	uid := c.GetCurrentUid()

	if err = a.checkAccount(c, uid, req.AccountId); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	trade := &models.InvestmentTrade{
		Uid:        uid,
		SecurityId: req.SecurityId,
		TradeType:  req.TradeType,
		TradeTime:  req.Time,
		Quantity:   quantity,
		Price:      price,
		Amount:     req.Amount,
		Fees:       req.Fees,
		AccountId:  req.AccountId,
		Source:     models.InvestmentTradeSourceManual,
		Comment:    req.Comment,
	}

	err = a.trades.CreateTrade(c, trade)

	if err != nil {
		log.Errorf(c, "[investments.TradeCreateHandler] failed to create trade for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.TradeCreateHandler] user \"uid:%d\" has created trade \"id:%d\" successfully", uid, trade.TradeId)

	return trade.ToInvestmentTradeInfoResponse(), nil
}

// TradeModifyHandler saves an existing trade for current user
func (a *InvestmentsApi) TradeModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentTradeModifyRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.TradeModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	quantity, qErr := parseOptionalScaled(req.Quantity)
	price, pErr := parseOptionalScaled(req.Price)

	if qErr != nil || pErr != nil {
		return nil, errs.ErrInvestmentTradeValueInvalid
	}

	uid := c.GetCurrentUid()

	if err = a.checkAccount(c, uid, req.AccountId); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	trade := &models.InvestmentTrade{
		TradeId:    req.Id,
		Uid:        uid,
		SecurityId: req.SecurityId,
		TradeType:  req.TradeType,
		TradeTime:  req.Time,
		Quantity:   quantity,
		Price:      price,
		Amount:     req.Amount,
		Fees:       req.Fees,
		AccountId:  req.AccountId,
		Comment:    req.Comment,
	}

	err = a.trades.ModifyTrade(c, trade)

	if err != nil {
		log.Errorf(c, "[investments.TradeModifyHandler] failed to update trade \"id:%d\" for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	saved, err := a.trades.GetTradeByTradeId(c, uid, req.Id)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.TradeModifyHandler] user \"uid:%d\" has updated trade \"id:%d\" successfully", uid, req.Id)

	return saved.ToInvestmentTradeInfoResponse(), nil
}

// TradeDeleteHandler deletes a trade, refusing when later sells would exceed holdings
func (a *InvestmentsApi) TradeDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentTradeDeleteRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.TradeDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.trades.DeleteTrade(c, uid, req.Id)

	if err != nil {
		log.Errorf(c, "[investments.TradeDeleteHandler] failed to delete trade \"id:%d\" for user \"uid:%d\", because %s", req.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.TradeDeleteHandler] user \"uid:%d\" has deleted trade \"id:%d\"", uid, req.Id)

	return true, nil
}

// TradeImportHandler previews or runs a CSV import
func (a *InvestmentsApi) TradeImportHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentTradeImportRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.TradeImportHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	if len(req.Csv) > maxInvestmentCsvBytes {
		return nil, errs.ErrInvestmentImportInvalid
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		log.Errorf(c, "[investments.TradeImportHandler] failed to get user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	rows, err := investments.ParseCSV(req.Csv, user.DefaultCurrency)

	if err != nil {
		if errors.Is(err, investments.ErrCsvEmpty) {
			return nil, errs.ErrInvestmentImportEmpty
		}

		log.Warnf(c, "[investments.TradeImportHandler] csv of user \"uid:%d\" is invalid, because %s", uid, err.Error())
		return nil, errs.ErrInvestmentImportInvalid
	}

	result, err := a.trades.ImportTrades(c, uid, models.InvestmentTradeSourceCsv, rows, req.DryRun)

	if err != nil {
		log.Errorf(c, "[investments.TradeImportHandler] failed to import trades for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.TradeImportHandler] user \"uid:%d\" import (dryRun:%t): %d created, %d skipped, %d failed", uid, req.DryRun, result.Created, result.Skipped, result.Failed)

	return result, nil
}

// TradeSyncHandler accepts an idempotent batch of externally sourced trades (broker phase)
func (a *InvestmentsApi) TradeSyncHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentTradeSyncRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[investments.TradeSyncHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	rows := make([]investments.RowResult, 0, len(req.Items))

	for index, item := range req.Items {
		line := index + 1
		quantity, qErr := parseOptionalScaled(item.Quantity)
		price, pErr := parseOptionalScaled(item.Price)

		if qErr != nil || pErr != nil {
			rows = append(rows, investments.RowResult{Line: line, Error: "quantity or price is not a valid decimal number"})
			continue
		}

		securityType := investments.SecurityType(item.SecurityType)

		if securityType == 0 {
			securityType = investments.SecurityTypeOther
		}

		name := item.Name

		if name == "" {
			name = item.Symbol
		}

		rows = append(rows, investments.RowResult{
			Line: line,
			Trade: &investments.ParsedTrade{
				Line:         line,
				Symbol:       item.Symbol,
				Name:         name,
				SecurityType: securityType,
				Currency:     item.Currency,
				Type:         investments.TradeType(item.TradeType),
				Time:         item.Time,
				Quantity:     quantity,
				Price:        price,
				Amount:       item.Amount,
				Fees:         item.Fees,
				ExternalID:   item.ExternalId,
				Comment:      item.Comment,
			},
		})
	}

	uid := c.GetCurrentUid()
	result, err := a.trades.ImportTrades(c, uid, models.InvestmentTradeSourceBroker, rows, req.DryRun)

	if err != nil {
		log.Errorf(c, "[investments.TradeSyncHandler] failed to sync trades for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[investments.TradeSyncHandler] user \"uid:%d\" sync (dryRun:%t): %d created, %d skipped, %d failed", uid, req.DryRun, result.Created, result.Skipped, result.Failed)

	return result, nil
}

// SummaryHandler returns holdings, totals and allocation in the user's default currency
func (a *InvestmentsApi) SummaryHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		log.Errorf(c, "[investments.SummaryHandler] failed to get user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	base := user.DefaultCurrency

	// exchange rates are optional: without them, single-currency holdings still work and others are flagged
	ratesAvailable := true
	rateResp, err := exchangerates.Container.GetLatestExchangeRates(c, uid, a.CurrentConfig())

	if err != nil {
		log.Warnf(c, "[investments.SummaryHandler] exchange rates are not available for user \"uid:%d\", because %s", uid, err.Error())
		ratesAvailable = false
		rateResp = nil
	}

	rates := services.RatesFromLatest(rateResp, base)

	summary, err := a.trades.BuildSummary(c, uid, base, rates)

	if err != nil {
		log.Errorf(c, "[investments.SummaryHandler] failed to build summary for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return models.ToInvestmentSummaryResponse(summary, ratesAvailable), nil
}

// PriceRefreshHandler refreshes prices of all non-manual securities now
func (a *InvestmentsApi) PriceRefreshHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	results, err := a.prices.RefreshPrices(c, uid)

	if err != nil {
		log.Errorf(c, "[investments.PriceRefreshHandler] failed to refresh prices for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrInvestmentPriceRefreshFailed)
	}

	return &models.SecurityPriceRefreshResponse{Results: results}, nil
}

// HistoryHandler returns the portfolio value over time in the user's default currency.
// Values before a security has a recorded price use its cost basis and are flagged as estimated.
func (a *InvestmentsApi) HistoryHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.InvestmentHistoryRequest
	err := c.ShouldBindQuery(&req)

	if err != nil {
		log.Warnf(c, "[investments.HistoryHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		log.Errorf(c, "[investments.HistoryHandler] failed to get user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	base := user.DefaultCurrency

	ratesAvailable := true
	rateResp, err := exchangerates.Container.GetLatestExchangeRates(c, uid, a.CurrentConfig())

	if err != nil {
		log.Warnf(c, "[investments.HistoryHandler] exchange rates are not available for user \"uid:%d\", because %s", uid, err.Error())
		ratesAvailable = false
		rateResp = nil
	}

	rates := services.RatesFromLatest(rateResp, base)

	stepDays := req.StepDays

	if stepDays <= 0 {
		stepDays = 7
	}

	points, missing, firstTrade, err := a.trades.BuildHistory(c, uid, base, rates, req.StartTime, time.Now().Unix(), int64(stepDays)*86400)

	if err != nil {
		log.Errorf(c, "[investments.HistoryHandler] failed to build history for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return &models.InvestmentHistoryResponse{
		BaseCurrency:          base,
		RatesAvailable:        ratesAvailable,
		UnconvertedCurrencies: missing,
		FirstTradeTime:        firstTrade,
		Points:                models.ToInvestmentHistoryPoints(points),
	}, nil
}
