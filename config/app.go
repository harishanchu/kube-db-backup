package config

import (
	"os"
	"log"
)

type AppConfig struct {
	LogLevel    string `json:"log_level"`
	Port        int    `json:"port"`
	ConfigPath  string `json:"config_path"`
	StoragePath string `json:"storage_path"`
	TmpPath     string `json:"tmp_path"`
	DataPath    string `json:"data_path"`
}

var AppEnv string

func init () {
	AppEnv = os.Getenv("APP_ENV")

	if AppEnv == "development" {
		log.Println("Running app dev mode")
	} else {
		log.Println("Running app in productoin mode")
	}

}