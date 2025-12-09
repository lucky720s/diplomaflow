package config

import (
	"strings"

	"github.com/spf13/viper"
)

func Load(path string, cfg interface{}) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return viper.Unmarshal(cfg)
}
