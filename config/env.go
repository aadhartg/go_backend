package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfigInterface struct {
	Environment string

	// Server
	ServerPort string

	// Database
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	JWTSecret          string
	AllowedOrigins     string
	SmtpFromEmail      string
	SmtpEmail          string
	SmtpPassword       string
	SmtpServer         string
	SmtpPort           string
	AppUrl             string
	ApiUrl             string
	EncryptionKey1     string
	EncryptionKey2     string
	PaypalClientID     string
	PaypalClientSecret string
	PaypalAPIUrl       string
	ClientEmail        string
	RedisHost          string
	RedisPort          string
	RedisPassword      string
	RedisDB            string
}

var AppConfig AppConfigInterface

// LoadEnv loads the environment variables from the .env file
func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	AppConfig.Environment = os.Getenv("ENVIRONMENT")

	// Load the environment variables based on the environment
	switch AppConfig.Environment {
	case "development":
		loadDevelopmentEnvVars()
	case "production":
		loadProductionEnvVars()
	case "testing":
		loadTestingEnvVars()
	case "localhost":
		loadLocalhostEnvVars()
	default:
		loadLocalhostEnvVars()
	}
}

// loadDevelopmentEnvVars loads the environment variables for the development environment
func loadDevelopmentEnvVars() {
	AppConfig.DBHost = os.Getenv("DEV_DB_HOST")
	AppConfig.DBPort = os.Getenv("DEV_DB_PORT")
	AppConfig.DBUser = os.Getenv("DEV_DB_USER")
	AppConfig.DBPassword = os.Getenv("DEV_DB_PASSWORD")
	AppConfig.DBName = os.Getenv("DEV_DB_NAME")
	AppConfig.ServerPort = os.Getenv("DEV_SERVER_PORT")
	AppConfig.JWTSecret = os.Getenv("DEV_JWT_SECRET_KEY")
	AppConfig.AllowedOrigins = os.Getenv("DEV_ALLOWED_ORIGINS")
	AppConfig.SmtpFromEmail = os.Getenv("DEV_SMTP_FROM_EMAIL")
	AppConfig.SmtpEmail = os.Getenv("DEV_SMTP_EMAIL")
	AppConfig.SmtpPassword = os.Getenv("DEV_SMTP_PASSWORD")
	AppConfig.SmtpServer = os.Getenv("DEV_SMTP_SERVER")
	AppConfig.SmtpPort = os.Getenv("DEV_SMTP_PORT")
	AppConfig.AppUrl = os.Getenv("DEV_APP_URL")
	AppConfig.ApiUrl = os.Getenv("DEV_API_URL")
	AppConfig.EncryptionKey1 = os.Getenv("DEV_ENCRYPTION_KEY1")
	AppConfig.EncryptionKey2 = os.Getenv("DEV_ENCRYPTION_KEY2")
	AppConfig.PaypalClientID = os.Getenv("DEV_PAYPAL_CLIENT_ID")
	AppConfig.PaypalClientSecret = os.Getenv("DEV_PAYPAL_CLIENT_SECRET")
	AppConfig.PaypalAPIUrl = os.Getenv("DEV_PAYPAL_API_URL")
	AppConfig.ClientEmail = os.Getenv("DEV_CLIENT_EMAIL")
	AppConfig.RedisHost = os.Getenv("DEV_REDIS_HOST")
	AppConfig.RedisPort = os.Getenv("DEV_REDIS_PORT")
	AppConfig.RedisPassword = os.Getenv("DEV_REDIS_PASSWORD")
	AppConfig.RedisDB = os.Getenv("DEV_REDIS_DB")
}

// loadProductionEnvVars loads the environment variables for the production environment
func loadProductionEnvVars() {
	AppConfig.DBHost = os.Getenv("PROD_DB_HOST")
	AppConfig.DBPort = os.Getenv("PROD_DB_PORT")
	AppConfig.DBUser = os.Getenv("PROD_DB_USER")
	AppConfig.DBPassword = os.Getenv("PROD_DB_PASSWORD")
	AppConfig.DBName = os.Getenv("PROD_DB_NAME")
	AppConfig.ServerPort = os.Getenv("PROD_SERVER_PORT")
	AppConfig.JWTSecret = os.Getenv("PROD_JWT_SECRET_KEY")
	AppConfig.AllowedOrigins = os.Getenv("PROD_ALLOWED_ORIGINS")
	AppConfig.SmtpFromEmail = os.Getenv("PROD_SMTP_FROM_EMAIL")
	AppConfig.SmtpEmail = os.Getenv("PROD_SMTP_EMAIL")
	AppConfig.SmtpPassword = os.Getenv("PROD_SMTP_PASSWORD")
	AppConfig.SmtpServer = os.Getenv("PROD_SMTP_SERVER")
	AppConfig.SmtpPort = os.Getenv("PROD_SMTP_PORT")
	AppConfig.AppUrl = os.Getenv("PROD_APP_URL")
	AppConfig.ApiUrl = os.Getenv("PROD_API_URL")
	AppConfig.EncryptionKey1 = os.Getenv("PROD_ENCRYPTION_KEY1")
	AppConfig.EncryptionKey2 = os.Getenv("PROD_ENCRYPTION_KEY2")
	AppConfig.PaypalClientID = os.Getenv("PROD_PAYPAL_CLIENT_ID")
	AppConfig.PaypalClientSecret = os.Getenv("PROD_PAYPAL_CLIENT_SECRET")
	AppConfig.PaypalAPIUrl = os.Getenv("PROD_PAYPAL_API_URL")
	AppConfig.ClientEmail = os.Getenv("PROD_CLIENT_EMAIL")
	AppConfig.RedisHost = os.Getenv("PROD_REDIS_HOST")
	AppConfig.RedisPort = os.Getenv("PROD_REDIS_PORT")
	AppConfig.RedisPassword = os.Getenv("PROD_REDIS_PASSWORD")
	AppConfig.RedisDB = os.Getenv("PROD_REDIS_DB")
}

// loadTestingEnvVars loads the environment variables for the testing environment
func loadTestingEnvVars() {
	AppConfig.DBHost = os.Getenv("TEST_DB_HOST")
	AppConfig.DBPort = os.Getenv("TEST_DB_PORT")
	AppConfig.DBUser = os.Getenv("TEST_DB_USER")
	AppConfig.DBPassword = os.Getenv("TEST_DB_PASSWORD")
	AppConfig.DBName = os.Getenv("TEST_DB_NAME")
	AppConfig.ServerPort = os.Getenv("TEST_SERVER_PORT")
	AppConfig.JWTSecret = os.Getenv("TEST_JWT_SECRET_KEY")
	AppConfig.AllowedOrigins = os.Getenv("TEST_ALLOWED_ORIGINS")
	AppConfig.SmtpFromEmail = os.Getenv("TEST_SMTP_FROM_EMAIL")
	AppConfig.SmtpEmail = os.Getenv("TEST_SMTP_EMAIL")
	AppConfig.SmtpPassword = os.Getenv("TEST_SMTP_PASSWORD")
	AppConfig.SmtpServer = os.Getenv("TEST_SMTP_SERVER")
	AppConfig.SmtpPort = os.Getenv("TEST_SMTP_PORT")
	AppConfig.AppUrl = os.Getenv("TEST_APP_URL")
	AppConfig.ApiUrl = os.Getenv("TEST_API_URL")
	AppConfig.EncryptionKey1 = os.Getenv("TEST_ENCRYPTION_KEY1")
	AppConfig.EncryptionKey2 = os.Getenv("TEST_ENCRYPTION_KEY2")
	AppConfig.PaypalClientID = os.Getenv("TEST_PAYPAL_CLIENT_ID")
	AppConfig.PaypalClientSecret = os.Getenv("TEST_PAYPAL_CLIENT_SECRET")
	AppConfig.PaypalAPIUrl = os.Getenv("TEST_PAYPAL_API_URL")
	AppConfig.ClientEmail = os.Getenv("TEST_CLIENT_EMAIL")
	AppConfig.RedisHost = os.Getenv("TEST_REDIS_HOST")
	AppConfig.RedisPort = os.Getenv("TEST_REDIS_PORT")
	AppConfig.RedisPassword = os.Getenv("TEST_REDIS_PASSWORD")
	AppConfig.RedisDB = os.Getenv("TEST_REDIS_DB")
}

// loadLocalhostEnvVars loads the environment variables for the localhost environment
func loadLocalhostEnvVars() {
	AppConfig.DBHost = os.Getenv("LOCAL_DB_HOST")
	AppConfig.DBPort = os.Getenv("LOCAL_DB_PORT")
	AppConfig.DBUser = os.Getenv("LOCAL_DB_USER")
	AppConfig.DBPassword = os.Getenv("LOCAL_DB_PASSWORD")
	AppConfig.DBName = os.Getenv("LOCAL_DB_NAME")
	AppConfig.ServerPort = os.Getenv("LOCAL_SERVER_PORT")
	AppConfig.JWTSecret = os.Getenv("LOCAL_JWT_SECRET_KEY")
	AppConfig.AllowedOrigins = os.Getenv("LOCAL_ALLOWED_ORIGINS")
	AppConfig.SmtpFromEmail = os.Getenv("LOCAL_SMTP_FROM_EMAIL")
	AppConfig.SmtpEmail = os.Getenv("LOCAL_SMTP_EMAIL")
	AppConfig.SmtpPassword = os.Getenv("LOCAL_SMTP_PASSWORD")
	AppConfig.SmtpServer = os.Getenv("LOCAL_SMTP_SERVER")
	AppConfig.SmtpPort = os.Getenv("LOCAL_SMTP_PORT")
	AppConfig.AppUrl = os.Getenv("LOCAL_APP_URL")
	AppConfig.ApiUrl = os.Getenv("LOCAL_API_URL")
	AppConfig.EncryptionKey1 = os.Getenv("LOCAL_ENCRYPTION_KEY1")
	AppConfig.EncryptionKey2 = os.Getenv("LOCAL_ENCRYPTION_KEY2")
	AppConfig.PaypalClientID = os.Getenv("LOCAL_PAYPAL_CLIENT_ID")
	AppConfig.PaypalClientSecret = os.Getenv("LOCAL_PAYPAL_CLIENT_SECRET")
	AppConfig.PaypalAPIUrl = os.Getenv("LOCAL_PAYPAL_API_URL")
	AppConfig.ClientEmail = os.Getenv("LOCAL_CLIENT_EMAIL")
	AppConfig.RedisHost = os.Getenv("LOCAL_REDIS_HOST")
	AppConfig.RedisPort = os.Getenv("LOCAL_REDIS_PORT")
	AppConfig.RedisPassword = os.Getenv("LOCAL_REDIS_PASSWORD")
	AppConfig.RedisDB = os.Getenv("LOCAL_REDIS_DB")
}
