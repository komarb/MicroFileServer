package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	DB   *DBConfig   		`json:"DbOptions"`
	Auth *AuthConfig 		`json:"AuthOptions"`
	App  *AppConfig  		`json:"AppOptions"`
}

type DBConfig struct {
	URI           	string 	`envconfig:"MFS_MONGO_URI",json:"uri"`
}
type AuthConfig struct {
	KeyURL   		string 	`envconfig:"MFS_AUTH_KEY_URL",json:"keyUrl"`
	Audience 		string 	`envconfig:"MFS_AUTH_AUDIENCE",json:"audience"`
	Issuer   		string 	`envconfig:"MFS_AUTH_ISSUER",json:"issuer"`
}
type AppConfig struct {
	AppPort  		string 	`envconfig:"MFS_APP_PORT",json:"appPort"`
	TestMode		bool	`envconfig:"MFS_TEST_MODE",json:"testMode"`
	MaxFileSize		int64	`envconfig:"MFS_MAX_FILE_SIZE",json:"maxFileSize"`
	PathPrefix		string	`envconfig:"MFS_PATH_PREFIX",json:"pathPrefix"`
}

func GetConfig() *Config {
	var config Config
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.WithFields(log.Fields{
			"function": "GetConfig.ReadFile",
			"error":    err,
		},
		).Warn("Can't read config.json file, shutting down...")
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.WithFields(log.Fields{
			"function": "GetConfig.Unmarshal",
			"error":    err,
		},
		).Warn("Can't correctly parse json from config.json, shutting down...")
	}

	data, err = ioutil.ReadFile("auth_config.json")
	if err != nil {
		log.WithFields(log.Fields{
			"function": "GetConfig.ReadFile",
			"error":    err,
		},
		).Warn("Can't read auth_config.json file, shutting down...")
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.WithFields(log.Fields{
			"function": "GetConfig.Unmarshal",
			"error":    err,
		},
		).Warning("Can't correctly parse json from auth_config.json, shutting down...")
	}

	err = envconfig.Process("mfs", &config)
	if err != nil {
		log.WithFields(log.Fields{
			"function": "envconfig.Process",
			"error":    err,
		},
		).Fatal("Can't read env vars, shutting down...")
	}

	return &config
}
