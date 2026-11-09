package services

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

func setupPreferenceTestDb(t *testing.T) core.Context {
	t.Helper()

	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      filepath.Join(t.TempDir(), "prefs.db"),
			MaxIdleConnection: 2,
			MaxOpenConnection: 4,
		},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
		UuidServerId:      1,
	}

	assert.NoError(t, datastore.InitializeDataStore(config))
	assert.NoError(t, uuid.InitializeUuidGenerator(config))
	assert.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.ExpbookPreference)))

	return core.NewNullContext()
}

func TestDisplayCurrencyDefaultsToNone(t *testing.T) {
	c := setupPreferenceTestDb(t)

	currency, err := ExpbookPreferences.GetDisplayCurrency(c, 1)
	assert.NoError(t, err)
	assert.Equal(t, "", currency)
}

func TestDisplayCurrencyIsStoredReplacedAndClearedPerUser(t *testing.T) {
	c := setupPreferenceTestDb(t)

	assert.NoError(t, ExpbookPreferences.SetDisplayCurrency(c, 1, " pln "))
	currency, _ := ExpbookPreferences.GetDisplayCurrency(c, 1)
	assert.Equal(t, "PLN", currency, "normalised to upper case")

	assert.NoError(t, ExpbookPreferences.SetDisplayCurrency(c, 1, "EUR"))
	currency, _ = ExpbookPreferences.GetDisplayCurrency(c, 1)
	assert.Equal(t, "EUR", currency, "second write replaces the first")

	other, _ := ExpbookPreferences.GetDisplayCurrency(c, 2)
	assert.Equal(t, "", other, "another user is not affected")

	assert.NoError(t, ExpbookPreferences.SetDisplayCurrency(c, 1, ""))
	currency, _ = ExpbookPreferences.GetDisplayCurrency(c, 1)
	assert.Equal(t, "", currency, "empty clears the choice")
}

func TestPreferenceRejectsInvalidUser(t *testing.T) {
	c := setupPreferenceTestDb(t)

	assert.Error(t, ExpbookPreferences.SetDisplayCurrency(c, 0, "PLN"))

	_, err := ExpbookPreferences.GetDisplayCurrency(c, 0)
	assert.Error(t, err)
}
