package handlers

type Store interface {
	GetByID(id int64) (*User, error)

	GetByEmail(email string) (*User, error)

	GetByContactNum(contactNum string) (*User, error)

	Insert(user *User) (*User, error)

	Delete(id int64) error

	// InsertTransaction(t *Transaction) error

	// DeleteTransaction(id int64) error

	// GetTransactions(uid string) (*[]Transaction, error)
}
