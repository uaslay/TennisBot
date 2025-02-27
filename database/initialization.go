package database

import (
	"fmt"
	"os"

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
		return nil
	}
	return db
}

// createDatabaseSchema is a function that creates the database schema
func (dbClient DBClient) createDatabaseSchema() {
	err := dbClient.DB.AutoMigrate(&Player{}, &ProposedGame{}, &DualGame{})
	if err != nil {
		panic(err)
	}
}

// InitDatabase is a function that initializes the database
func (dbClient *DBClient) InitDatabase() {
	dbConf := DBConfigs(os.Getenv("HOST"), os.Getenv("USER"), os.Getenv("PASSWORD"), os.Getenv("DBNAME"), os.Getenv("PORT"), os.Getenv("SSLMODE"))
	for {
		dbClient.DB = ConnectToDatabase(dbConf)
		if dbClient.DB != nil {
			break
		}
	}

	dbClient.createDatabaseSchema()
}
