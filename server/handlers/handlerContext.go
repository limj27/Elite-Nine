package handlers

import (
	"trivia-server/db"
	"trivia-server/sessions"
)

type HandlerContext struct {
	SingingKey string
	Sessions   sessions.Store
	Users      Store
	DB         db.MysqlStore
}
