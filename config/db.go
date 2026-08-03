package config

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConnection holds the database connection pool
var DBConnection *pgxpool.Pool

// ConnectDB establishes a connection to the PostgreSQL database
func ConnectDB(cfg *Config) {
	var err error
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	DBConnection, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Successfully connected to database")
}

// CloseDB closes the database connection
func CloseDB() {
	if DBConnection != nil {
		DBConnection.Close()
	}
}

