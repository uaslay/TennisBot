package database

import (
	"os"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBClient is a struct that contains a pointer to a gorm.DB instance
type DBClient struct {
	DB *gorm.DB
}

// Credentials is a struct that contains the database connection parameters
type Credentials struct {
	Host         string
	User         string
	Password     string
	DatabaseName string
	Port         string
	SSLMode      string
}

// DBConfigs is a function that returns a Credentials struct with the database connection parameters
func DBConfigs(host string, user string, password string, databaseName string, port string, sslMode string) Credentials {
	return Credentials{
		Host:         host,
		User:         user,
		Password:     password,
		DatabaseName: databaseName,
		Port:         port,
		SSLMode:      sslMode,
	}
}

// ConnectToDatabase is a function that connects to the database using the credentials provided
func ConnectToDatabase(credentials Credentials) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		credentials.Host,
		credentials.User,
		credentials.Password,
		credentials.DatabaseName,
		credentials.Port,
		credentials.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// Додамо логування помилки підключення
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		return nil
	}
	return db
}

// createDatabaseSchema is a function that creates the database schema
func (dbClient DBClient) createDatabaseSchema() {
    // 1. Спочатку мігруємо Player
    err := dbClient.DB.AutoMigrate(&Player{}) // Стек-трейс вказує на паніку тут
    if err != nil {
		log.Fatalf("Failed to migrate Player table: %v", err)
    }
    fmt.Println("Player table migrated successfully.")

    // 2. Потім мігруємо ProposedGame (залежить від Player)
    err = dbClient.DB.AutoMigrate(&ProposedGame{}) // Лог БД вказує на помилку тут
    if err != nil {
        log.Fatalf("Failed to migrate ProposedGame table: %v", err)
    }
    fmt.Println("ProposedGame table migrated successfully.")

    // 3. Потім мігруємо GameResponse (залежить від Player і ProposedGame) та DualGame
    err = dbClient.DB.AutoMigrate(&GameResponse{}, &DualGame{})
    if err != nil {
        log.Fatalf("Failed to migrate GameResponse/DualGame tables: %v", err)
    }
    fmt.Println("GameResponse and DualGame tables migrated successfully.")

    fmt.Println("Database schema migrated successfully.")
}

// InitDatabase is a function that initializes the database
func (dbClient *DBClient) InitDatabase() {
	dbConf := DBConfigs(os.Getenv("HOST"), os.Getenv("USER"), os.Getenv("PASSWORD"), os.Getenv("DBNAME"), os.Getenv("PORT"), os.Getenv("SSLMODE"))
	// Додамо спроби підключення з невеликою затримкою
	maxRetries := 5
	retryDelay := time.Second * 2
	for i := 0; i < maxRetries; i++ {
		dbClient.DB = ConnectToDatabase(dbConf)
		if dbClient.DB != nil {
			fmt.Println("Successfully connected to the database.")
			break
		}
		fmt.Fprintf(os.Stderr, "Database connection attempt %d failed. Retrying in %v...\n", i+1, retryDelay)
		time.Sleep(retryDelay)
	}

	if dbClient.DB == nil {
		log.Fatalf("Failed to connect to database after multiple retries.")
	}

	dbClient.createDatabaseSchema()
}
