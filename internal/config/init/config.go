package init

import (
	"github.com/spf13/viper"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/config"
	"github.com/wal-g/wal-g/internal/tracelog"
)

func init() {
	for _, adapter := range internal.StorageAdapters {
		allowedConfigKeys := []string{
			adapter.prefixName,
			config.ToWalgSettingName(adapter.prefixName),
		}
		config.UpdateAllowed(allowedConfigKeys)
		for _, settingName := range adapter.settingNames {
			allowedConfigKeys = []string{
				"WALG_" + settingName,
				"WALE_" + settingName,
				settingName,
			}
			config.UpdateAllowed(allowedConfigKeys)
		}
	}
	readConfig()
	verifyConfig()
}

func verifyConfig() {
	if WalgConfig == nil {
		return
	}
	for _, extension := range internal.Extensions {
		for key := range extension.GetAllowedConfigKeys() {
			config.UpdateAllowed([]string{key})
		}
	}
	for k := range *WalgConfig {
		if _, ok := config.CheckAllowed(k); !ok {
			tracelog.ErrorLogger.Panic("Settings " + k + " is unknown")
		}
	}
}

func readConfig() {
	cfg := make(map[string]string)
	WalgConfig = &cfg
	for _, key := range viper.AllKeys() {
		cfg[key] = viper.GetString(key)
	}
}