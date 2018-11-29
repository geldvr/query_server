package config

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"path"
	"runtime"
)

var (
	configPath = os.Getenv("ITV_CONFIG_PATH")

	BaseUrl   = ""
	MaxWorker = os.Getenv("MAX_WORKERS")
	MaxQueue  = os.Getenv("MAX_QUEUE")
)

func GetString(key string) string {
	if !viper.IsSet(key) {
		log.Fatalf("Key \"%s\" not found in config file", key)
	}
	return viper.GetString(key)
}

func GetInt(key string) int {
	if !viper.IsSet(key) {
		log.Fatalf("Key \"%s\" not found in config file", key)
	}
	return viper.GetInt(key)
}

func getConfigPath() string {
	if len(configPath) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		dir := path.Dir(filename)
		configPath = path.Join(dir, "../")
	}

	return path.Join(configPath, "config.json")
}

func readFile(filePath string) {
	viper.SetConfigFile(filePath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Config error: %s", err.Error())
	}
}

func init() {
	readFile(getConfigPath())
	BaseUrl = GetString("baseUrl")
}
