// config/config.go
package config

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the database connection, checks for the existence of the target database,
// and creates it if it doesn't exist.
func InitDB() {
	// Load environment variables
	LoadEnv()
	cfg := AppConfig

	// Construct the Data Source Name (DSN) for connecting to the default 'postgres' database
	defaultDsn := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBPort)

	// Open a connection to the default 'postgres' database
	defaultDB, err := gorm.Open(postgres.Open(defaultDsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Suppress default logging
	})
	if err != nil {
		log.Fatalf("Failed to connect to default database: %v", err)
	}

	// Check if the target database exists
	var exists bool
	query := "SELECT 1 FROM pg_database WHERE datname = ?"
	err = defaultDB.Raw(query, cfg.DBName).Scan(&exists).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if !exists {
		// Create the target database if it doesn't exist
		createDBQuery := fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)
		if err := defaultDB.Exec(createDBQuery).Error; err != nil {
			log.Fatalf("Failed to create database %s: %v", cfg.DBName, err)
		}
		log.Printf("Database %s created successfully", cfg.DBName)
	} else {
		log.Printf("Database %s already exists", cfg.DBName)
	}

	// Construct the DSN for connecting to the target database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	// Open a connection to the target database
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB instance: %v", err)
	}

	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(10)

	log.Printf("%s environment: %s", "Database connection established", cfg.Environment)
}

// CreateEnumTypes creates the necessary enum types in the database if they don't exist.
// The method takes a list of enum types and their corresponding values. It checks if
// the type exists using a transaction and if not, creates the type using a CREATE TYPE
// query. If the type already exists, it logs a message and skips the creation.
func CreateEnumTypes() error {
	enumTypes := map[string]string{
		"dna_payment_status":           "dna_payment_status",
		"dna_order_status":             "dna_order_status",
		"dna_notification_type":        "dna_notification_type",
		"dna_notification_entity_type": "dna_notification_entity_type",
		"dna_kit_status":               "dna_kit_status",
	}

	enumValues := map[string][]string{
		"dna_payment_status":           {"Pending", "Completed", "Failed"},
		"dna_order_status":             {"Pending", "Processing", "Dispatched", "Shipped", "Delivered", "Cancelled"},
		"dna_notification_type":        {"Admin Management", "Inventory Management", "Order Purchase", "Order Management"},
		"dna_notification_entity_type": {"barcodes", "customers", "invoices", "kits", "orders", "payments", "roles", "users"},
		"dna_kit_status":               {"Not-Received", "Received", "Reject", "Send"},
	}

	// Iterate over the enum types and create them if they don't exist
	for dbType, enumName := range enumTypes {
		// Check if type exists using a transaction to handle potential errors
		tx := DB.Begin()
		var exists bool
		checkQuery := `
			SELECT EXISTS (
				SELECT 1 FROM pg_type 
				WHERE typname = $1
			);
		`
		if err := tx.Raw(checkQuery, dbType).Scan(&exists).Error; err != nil {
			tx.Rollback()
			log.Printf("Error checking enum %s: %v", dbType, err)
			continue // Skip this enum and continue with others
		}

		if !exists {
			// Construct the CREATE TYPE query with values
			values := strings.Join(enumValues[enumName], "', '")
			createQuery := fmt.Sprintf("CREATE TYPE %s AS ENUM ('%s')", dbType, values)

			if err := tx.Exec(createQuery).Error; err != nil {
				tx.Rollback()
				// Check if the error is because the type already exists (race condition)
				if strings.Contains(err.Error(), "already exists") {
					log.Printf("Enum type %s already exists (created by another process)", dbType)
					continue
				}
				log.Printf("Error creating enum %s: %v", dbType, err)
				continue // Skip this enum and continue with others
			}
			log.Printf("Successfully created enum type: %s", dbType)
		} else {
			log.Printf("Enum type %s already exists, skipping", dbType)
		}

		tx.Commit()
	}

	return nil
}
