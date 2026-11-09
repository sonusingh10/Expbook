package services

import (
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// ExpbookPreferenceService stores small per-user preferences (Expbook)
type ExpbookPreferenceService struct {
	ServiceUsingDB
}

// Initialize an Expbook preference service singleton instance
var (
	ExpbookPreferences = &ExpbookPreferenceService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// GetPreference returns the stored value, or "" when the user never set one
func (s *ExpbookPreferenceService) GetPreference(c core.Context, uid int64, key string) (string, error) {
	if uid <= 0 {
		return "", errs.ErrUserIdInvalid
	}

	preference := &models.ExpbookPreference{}
	has, err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND pref_key=?", uid, key).Get(preference)

	if err != nil {
		return "", err
	}

	if !has {
		return "", nil
	}

	return preference.PrefValue, nil
}

// SetPreference stores the value, replacing an earlier one. An empty value is stored as "" (meaning "not chosen").
func (s *ExpbookPreferenceService) SetPreference(c core.Context, uid int64, key string, value string) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		exists, err := sess.Where("uid=? AND pref_key=?", uid, key).Exist(&models.ExpbookPreference{})

		if err != nil {
			return err
		}

		now := time.Now().Unix()

		if exists {
			_, err = sess.Where("uid=? AND pref_key=?", uid, key).Cols("pref_value", "updated_unix_time").Update(&models.ExpbookPreference{PrefValue: value, UpdatedUnixTime: now})
			return err
		}

		_, err = sess.Insert(&models.ExpbookPreference{Uid: uid, PrefKey: key, PrefValue: value, UpdatedUnixTime: now})

		return err
	})
}

// GetDisplayCurrency returns the chosen display currency, "" when none is chosen
func (s *ExpbookPreferenceService) GetDisplayCurrency(c core.Context, uid int64) (string, error) {
	return s.GetPreference(c, uid, models.ExpbookPreferenceDisplayCurrency)
}

// SetDisplayCurrency stores the display currency; "" clears it
func (s *ExpbookPreferenceService) SetDisplayCurrency(c core.Context, uid int64, currency string) error {
	return s.SetPreference(c, uid, models.ExpbookPreferenceDisplayCurrency, strings.ToUpper(strings.TrimSpace(currency)))
}
