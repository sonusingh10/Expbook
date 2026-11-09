package models

// ExpbookPreference is one per-user preference stored on the server so it follows the user across devices (Expbook).
// It is deliberately separate from the upstream application cloud settings, which are opt-in and have a fixed key list.
type ExpbookPreference struct {
	Uid             int64  `xorm:"PK NOT NULL"`
	PrefKey         string `xorm:"PK VARCHAR(64) NOT NULL"`
	PrefValue       string `xorm:"VARCHAR(255) NOT NULL"`
	UpdatedUnixTime int64
}

// Keys of Expbook preferences
const (
	ExpbookPreferenceDisplayCurrency = "displayCurrency"
)

// DisplayCurrencyUpdateRequest represents the request to change the display currency.
// An empty currency clears the choice, so screens go back to the profile currency.
type DisplayCurrencyUpdateRequest struct {
	Currency string `json:"currency" binding:"omitempty,len=3,validCurrency"`
}

// DisplayCurrencyResponse is the stored display currency ("" when none is chosen)
type DisplayCurrencyResponse struct {
	Currency string `json:"currency"`
}
