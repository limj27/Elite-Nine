package handlers

import (
	"errors"
)

var ErrUserNotFound = errors.New("user not found")

type Store interface {
	GetByID(id int) (User, error)

	Insert(user User) (User, error)

	Update(id int64, updates Updates) (User, error)

	Delete(id int64) error
}
