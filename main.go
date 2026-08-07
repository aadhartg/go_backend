package main

import (
	"log"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"theransticslabs/m/config"
	"theransticslabs/m/models"
	"theransticslabs/m/routes"
	"theransticslabs/m/seeds"
	"theransticslabs/m/services"
	"theransticslabs/m/utils"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// Set Gin mode based on environment
	if config.AppConfig.Environment == "production" || config.AppConfig.Environment == "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.Default()

	// Configure CORS
	allowedOrigins := strings.Split(config.AppConfig.AllowedOrigins, ",")
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Consent-Token"},
		AllowCredentials: true,
	}))

	// Initialize the Database
	config.InitDB()
	// Initialize Redis
	config.RedisClient = config.InitRedis()

	if err := config.RedisClient.Ping(config.Ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis Connected")

	// Create enum types
	if err := config.CreateEnumTypes(); err != nil {
		log.Fatalf("Failed to create enum types: %v", err)
	}

	// Auto-Migrate the models
	err := config.DB.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.Kit{},
		&models.Customer{},
		&models.Order{},
		&models.Payment{},
		&models.Invoice{},
		&models.ErrorLog{},
		&models.Barcode{},
		&models.Notification{},
		&models.QuantityDiscounts{},
		&models.KitRegistration{},
		&models.Lab{},
		&models.Video{},
		&models.Research{},
		&models.ResearchCategory{},
		&models.ResearchDocument{},
		&models.DeclarationForm{},
		&models.Declaration{},
		&models.EConsentForm{},
		&models.ConsentDeclaration{},
		&models.EConsentAssignment{},
		&models.DeclarationResponse{},
		&models.PublishedResearch{},
		&models.PatientEConsent{},
		&models.EConsentFormDeclarations{},
		&models.EConsentFormVersion{},
		&models.Genomics{},
		&models.ConsentSession{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println(utils.MsgDatabaseMigrated)

	// Run the Seeders
	seeds.SeedAll()

	// Initialize Routes
	routes.SetupRoutes(router)

	// Serve static files
	router.Static("/images", "./public/images")
	router.Static("/files", "./public/files")

	// Start background services
	go services.LogCleanupJob(config.DB)
	// Start the Server
	log.Printf(utils.MsgServerStarted, config.AppConfig.ServerPort)
	if err := router.Run(":" + config.AppConfig.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
