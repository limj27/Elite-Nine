package handlers

import (
	"fmt"
	"net/mail"

	"golang.org/x/crypto/bcrypt"
)

var bcryptCost = 13

// Struct to store information about the users. The json tags are also included
// so that these structs can be used for data transfer between the client and the api.
type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	PassHash []byte `json:"-"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type NewUser struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	PasswordConf string `json:"passwordConf"`
}

type Updates struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Validates the new user and returns error if the rules fail
func (nu *NewUser) Validate() error {
	email, err := mail.ParseAddress(nu.Email)
	if err != nil {
		return fmt.Errorf("Invalid email address %v", email)
	}
	if len(nu.Password) < 6 {
		return fmt.Errorf("Password %v is less than 6 characters", nu.Password)
	}
	if nu.Password != nu.PasswordConf {
		return fmt.Errorf("Password doesn't match confirmed password")
	}
	if len(nu.Name) <= 0 {
		return fmt.Errorf("Provide a name")
	}
	return nil
}

// ToUser converts NewUser to a User
func (nu *NewUser) ToUser() (*User, error) {
	err := nu.Validate()
	if err != nil {
		return nil, err
	}
	user := &User{}
	user.Email = nu.Email
	user.ID = 0
	user.Name = nu.Name
	user.SetPassword(nu.Password)
	return user, nil
}

// Sets the user entered password into a hashed password in the User struct
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}
	u.PassHash = hashedPassword
	return nil
}

// When a user logs in, it compares the plaintext password against the store hash and returns the error
func (u *User) Authenticate(password string) error {
	err := bcrypt.CompareHashAndPassword(u.PassHash, []byte(password))
	if err != nil {
		return err
	}
	return nil
}

func (u *User) ApplyUpdates(updates *Updates) error {
	if updates.Name == "" {
		return fmt.Errorf("No new name provided, sticking with existing")
	}
	u.Name = updates.Name
	return nil
}
