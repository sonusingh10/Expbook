package services

import (
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// SecurityService represents security service (Expbook)
type SecurityService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a security service singleton instance
var (
	Securities = &SecurityService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// NormalizeSymbol trims and upper-cases a symbol so lookups are case-insensitive
func NormalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

// GetAllSecuritiesByUid returns all non-deleted securities of the user
func (s *SecurityService) GetAllSecuritiesByUid(c core.Context, uid int64) ([]*models.Security, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var securities []*models.Security
	err := s.UserDataDB(uid).NewSession(c).Where("uid=? AND deleted=?", uid, false).OrderBy("symbol asc").Find(&securities)

	return securities, err
}

// GetSecurityBySecurityId returns one security of the user
func (s *SecurityService) GetSecurityBySecurityId(c core.Context, uid int64, securityId int64) (*models.Security, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if securityId <= 0 {
		return nil, errs.ErrSecurityIdInvalid
	}

	security := &models.Security{}
	has, err := s.UserDataDB(uid).NewSession(c).ID(securityId).Where("uid=? AND deleted=?", uid, false).Get(security)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrSecurityNotFound
	}

	return security, nil
}

func validateSecurityFields(security *models.Security) error {
	if security.SecurityType < 1 || security.SecurityType > 5 {
		return errs.ErrSecurityTypeInvalid
	}

	if security.PriceSource == "" {
		security.PriceSource = models.SecurityPriceSourceManual
	}

	if !models.IsValidSecurityPriceSource(security.PriceSource) {
		return errs.ErrSecurityPriceSourceInvalid
	}

	security.Symbol = NormalizeSymbol(security.Symbol)
	security.Name = strings.TrimSpace(security.Name)
	security.Currency = strings.ToUpper(strings.TrimSpace(security.Currency))
	security.PriceSymbol = strings.TrimSpace(security.PriceSymbol)

	if security.PriceSource != models.SecurityPriceSourceManual && security.PriceSymbol == "" {
		security.PriceSymbol = security.Symbol
	}

	return nil
}

// CreateSecurity saves a new security model to database
func (s *SecurityService) CreateSecurity(c core.Context, security *models.Security) error {
	if security.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if err := validateSecurityFields(security); err != nil {
		return err
	}

	return s.UserDataDB(security.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		return s.createSecurityInSession(sess, security)
	})
}

func (s *SecurityService) createSecurityInSession(sess datastore.DatabaseSession, security *models.Security) error {
	exists, err := sess.Cols("uid", "deleted", "symbol").Where("uid=? AND deleted=? AND symbol=?", security.Uid, false, security.Symbol).Limit(1).Exist(&models.Security{})

	if err != nil {
		return err
	} else if exists {
		return errs.ErrSecuritySymbolAlreadyExists
	}

	security.SecurityId = s.GenerateUuid(uuid.UUID_TYPE_SECURITY)

	if security.SecurityId < 1 {
		return errs.ErrSystemIsBusy
	}

	now := time.Now().Unix()
	security.Deleted = false
	security.CreatedUnixTime = now
	security.UpdatedUnixTime = now

	_, err = sess.Insert(security)

	return err
}

// ModifySecurity saves an existed security model to database
func (s *SecurityService) ModifySecurity(c core.Context, security *models.Security) error {
	if security.Uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if err := validateSecurityFields(security); err != nil {
		return err
	}

	security.UpdatedUnixTime = time.Now().Unix()

	return s.UserDataDB(security.Uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		exists, err := sess.Cols("uid", "deleted", "symbol", "security_id").Where("uid=? AND deleted=? AND symbol=? AND security_id<>?", security.Uid, false, security.Symbol, security.SecurityId).Limit(1).Exist(&models.Security{})

		if err != nil {
			return err
		} else if exists {
			return errs.ErrSecuritySymbolAlreadyExists
		}

		// changing the currency would silently re-denominate recorded trades
		current := &models.Security{}
		found, err := sess.Where("uid=? AND deleted=? AND security_id=?", security.Uid, false, security.SecurityId).Get(current)

		if err != nil {
			return err
		} else if !found {
			return errs.ErrSecurityNotFound
		}

		if current.Currency != security.Currency {
			hasTrades, err := sess.Cols("uid", "deleted", "security_id").Where("uid=? AND deleted=? AND security_id=?", security.Uid, false, security.SecurityId).Limit(1).Exist(&models.InvestmentTrade{})

			if err != nil {
				return err
			} else if hasTrades {
				return errs.ErrSecurityCurrencyCannotBeChanged
			}
		}

		updatedRows, err := sess.ID(security.SecurityId).Cols("name", "symbol", "security_type", "currency", "price_source", "price_symbol", "comment", "updated_unix_time").Where("uid=? AND deleted=?", security.Uid, false).Update(security)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrSecurityNotFound
		}

		return nil
	})
}

// SetLatestPrice stores a new latest price (scaled 1e8) and records it in the price history
func (s *SecurityService) SetLatestPrice(c core.Context, uid int64, securityId int64, price int64, priceTime int64, source string) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	if price <= 0 {
		return errs.ErrSecurityPriceInvalid
	}

	if priceTime <= 0 {
		priceTime = time.Now().Unix()
	}

	priceId := s.GenerateUuid(uuid.UUID_TYPE_SECURITY_PRICE)

	if priceId < 1 {
		return errs.ErrSystemIsBusy
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		updateModel := &models.Security{
			LatestPrice:     price,
			LatestPriceTime: priceTime,
			UpdatedUnixTime: time.Now().Unix(),
		}

		updatedRows, err := sess.ID(securityId).Cols("latest_price", "latest_price_time", "updated_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrSecurityNotFound
		}

		_, err = sess.Insert(&models.SecurityPrice{
			PriceId:    priceId,
			Uid:        uid,
			SecurityId: securityId,
			PriceTime:  priceTime,
			Price:      price,
			Source:     source,
		})

		return err
	})
}

// DeleteSecurity soft-deletes a security; refused while it still has trades
func (s *SecurityService) DeleteSecurity(c core.Context, uid int64, securityId int64) error {
	if uid <= 0 {
		return errs.ErrUserIdInvalid
	}

	updateModel := &models.Security{
		Deleted:         true,
		DeletedUnixTime: time.Now().Unix(),
	}

	return s.UserDataDB(uid).DoTransaction(c, func(sess datastore.DatabaseSession) error {
		inUse, err := sess.Cols("uid", "deleted", "security_id").Where("uid=? AND deleted=? AND security_id=?", uid, false, securityId).Limit(1).Exist(&models.InvestmentTrade{})

		if err != nil {
			return err
		} else if inUse {
			return errs.ErrSecurityInUseCannotBeDeleted
		}

		deletedRows, err := sess.ID(securityId).Cols("deleted", "deleted_unix_time").Where("uid=? AND deleted=?", uid, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrSecurityNotFound
		}

		return nil
	})
}
