package db

import "database/sql"

type MysqlStore struct {
	db *sql.DB
}
