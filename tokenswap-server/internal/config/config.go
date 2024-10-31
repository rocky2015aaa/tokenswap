package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	Version = "Dev"
	Build   = "Dev"
	Date    = "Dev"
)

const (
	EnvStsvrJwtSecret  = "STSVR_JWT_SECRET"
	EnvStsvrPort       = "STSVR_BACKEND_PORT"
	EnvStsvrLogLevel   = "STSVR_LOG_LEVEL"
	EnvStsvrMongodbUri = "STSVR_MONGODB_URI"
	EnvStsvrGinMode    = "STSVR_GIN_MODE"
	EnvStsvrProfile    = "STSVR_PROFILE"

	EnvFile = ".env"
)

func init() {
	fmt.Printf("Build Date: %s\nBuild Version: %s\nBuild: %s\n\n", Date, Version, Build)
	envProfile := os.Getenv(EnvStsvrProfile)
	if envProfile != "" {
		envProfile = fmt.Sprintf("_%s", envProfile)
	}
	err := godotenv.Load(EnvFile + envProfile)
	if err != nil {
		log.Fatalf("Error loading %s file: %v", EnvFile, err)
	}
	logLevel, err := log.ParseLevel(os.Getenv(EnvStsvrLogLevel))
	if err != nil {
		logLevel = log.DebugLevel
	}
	log.SetLevel(logLevel)
	log.SetFormatter(&log.JSONFormatter{})
}
