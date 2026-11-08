package errs

import "net/http"

// Error codes related to investments (Expbook)
var (
	ErrSecurityIdInvalid             = NewNormalError(NormalSubcategoryInvestment, 0, http.StatusBadRequest, "security id is invalid")
	ErrSecurityNotFound              = NewNormalError(NormalSubcategoryInvestment, 1, http.StatusBadRequest, "security not found")
	ErrSecuritySymbolAlreadyExists   = NewNormalError(NormalSubcategoryInvestment, 2, http.StatusBadRequest, "security symbol already exists")
	ErrSecurityInUseCannotBeDeleted  = NewNormalError(NormalSubcategoryInvestment, 3, http.StatusBadRequest, "security has trades and cannot be deleted")
	ErrSecurityTypeInvalid           = NewNormalError(NormalSubcategoryInvestment, 4, http.StatusBadRequest, "security type is invalid")
	ErrSecurityPriceSourceInvalid    = NewNormalError(NormalSubcategoryInvestment, 5, http.StatusBadRequest, "security price source is invalid")
	ErrSecurityPriceInvalid          = NewNormalError(NormalSubcategoryInvestment, 6, http.StatusBadRequest, "security price is invalid")
	ErrInvestmentTradeIdInvalid      = NewNormalError(NormalSubcategoryInvestment, 7, http.StatusBadRequest, "investment trade id is invalid")
	ErrInvestmentTradeNotFound       = NewNormalError(NormalSubcategoryInvestment, 8, http.StatusBadRequest, "investment trade not found")
	ErrInvestmentTradeTypeInvalid    = NewNormalError(NormalSubcategoryInvestment, 9, http.StatusBadRequest, "investment trade type is invalid")
	ErrInvestmentTradeValueInvalid   = NewNormalError(NormalSubcategoryInvestment, 10, http.StatusBadRequest, "trade quantity, price, fees or amount is invalid")
	ErrInvestmentSellExceedsHoldings = NewNormalError(NormalSubcategoryInvestment, 11, http.StatusBadRequest, "sell quantity exceeds the quantity held at that date")
	ErrInvestmentImportEmpty         = NewNormalError(NormalSubcategoryInvestment, 12, http.StatusBadRequest, "import content is empty")
	ErrInvestmentImportInvalid       = NewNormalError(NormalSubcategoryInvestment, 13, http.StatusBadRequest, "import content is invalid")
	ErrInvestmentPriceRefreshFailed  = NewNormalError(NormalSubcategoryInvestment, 14, http.StatusBadRequest, "price refresh failed")
)
