package handlers

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	PassHash []byte `json:"-"`
	Stats    struct {
		Wins   int `json:"wins"`
		Losses int `json:"losses"`
	} `json:"stats"`
}

type NewUser struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	PasswordConf string `json:"passwordConfirm"`
}

func (u *NewUser) IsValid() error {
	if u.Username == "" || u.Password == "" {
		return fmt.Errorf("Invalid Username or Password")
	}

	if len(u.Password) < 8 {
		return fmt.Errorf("Password must be at least 8 characters long")
	}

	if u.Password != u.PasswordConf {
		return fmt.Errorf("Password and Password Confirmation do not match")
	}

	return nil
}

func (u *NewUser) ToUser() (*User, error) {
	err := u.IsValid()
	if err != nil {
		return nil, err
	}
	user := &User{}
	user.Username = u.Username
	user.HashPassword(u.Password)
	user.Stats.Wins = 0
	user.Stats.Losses = 0
	user.ID = fmt.Sprintf("%s-%d", user.Username, user.Stats.Wins+user.Stats.Losses) // Example ID generation
	return user, nil
}

func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PassHash = hashedPassword
	return nil
}

func (u *User) Authenticate(password string) error {
	err := bcrypt.CompareHashAndPassword(u.PassHash, []byte(password))
	if err != nil {
		return err
	}
	return nil
}
