package storage

import "database/sql"

// MySQLClient wraps direct SQL access for triggers and event logs.
type MySQLClient struct {
	db *sql.DB
}

// NewMySQLClient wires a sql.DB; pass a configured instance from main.
func NewMySQLClient(db *sql.DB) *MySQLClient {
	return &MySQLClient{db: db}
}
