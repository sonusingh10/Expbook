package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// ExpbookPreferencesApi represents the Expbook preferences api (Expbook)
type ExpbookPreferencesApi struct {
	ApiUsingConfig
	preferences *services.ExpbookPreferenceService
}

// Initialize an Expbook preferences api singleton instance
var (
	ExpbookPreferences = &ExpbookPreferencesApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		preferences: services.ExpbookPreferences,
	}
)

// DisplayCurrencyGetHandler returns the display currency chosen by the current user ("" when none)
func (a *ExpbookPreferencesApi) DisplayCurrencyGetHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	currency, err := a.preferences.GetDisplayCurrency(c, uid)

	if err != nil {
		log.Errorf(c, "[expbook_preferences.DisplayCurrencyGetHandler] failed to get display currency for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return &models.DisplayCurrencyResponse{Currency: currency}, nil
}

// DisplayCurrencyUpdateHandler stores the display currency chosen by the current user
func (a *ExpbookPreferencesApi) DisplayCurrencyUpdateHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.DisplayCurrencyUpdateRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[expbook_preferences.DisplayCurrencyUpdateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.preferences.SetDisplayCurrency(c, uid, req.Currency)

	if err != nil {
		log.Errorf(c, "[expbook_preferences.DisplayCurrencyUpdateHandler] failed to set display currency for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[expbook_preferences.DisplayCurrencyUpdateHandler] user \"uid:%d\" display currency set to \"%s\"", uid, req.Currency)

	return &models.DisplayCurrencyResponse{Currency: req.Currency}, nil
}
