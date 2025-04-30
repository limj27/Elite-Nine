package handlers

import "database/sql"

type MysqlStore struct {
	DB *sql.DB
}
