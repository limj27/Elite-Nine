package handlers

import "trivia-server/sessions"

type HandlerContext struct {
	SigningKey string
	Sessions   sessions.Store
	Users      Store
	DB         MysqlStore
}
