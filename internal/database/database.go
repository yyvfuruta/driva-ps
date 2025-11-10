// Package database provides a function for connecting to the database.
package database

import (
	"database/sql"
	"fmt"
	"os"
)

func NewConnection() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	userName := os.Getenv("DB_USER_NAME")
	userPass := os.Getenv("DB_USER_PASS")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSLMODE")

	envVars := map[string]string{
		"DB_HOST":      host,
		"DB_PORT":      port,
		"DB_USER_NAME": userName,
		"DB_USER_PASS": userPass,
		"DB_NAME":      dbName,
	}

	for key, value := range envVars {
		if value == "" {
			return nil, fmt.Errorf("%s environment variable not set", key)
		}
	}

	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, userName, userPass, dbName, sslMode,
	)

	c, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := c.Ping(); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}
