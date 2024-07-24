package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"strings"
	"time"
)

type DBModel struct {
	DB *sql.DB
}

type Models struct {
	DB DBModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		DB: DBModel{DB: db},
	}
}

type Widget struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	IsRecurring    string    `json:"is_recurring"`
	PlanID         string    `json:"plan_id"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

type Order struct {
	ID            int       `json:"id"`
	WidgetID      int       `json:"widget_id"`
	TransactionID int       `json:"transaction_id"`
	CustomerID    int       `json:"customer_id"`
	StatusID      int       `json:"status_id"`
	Quantity      int       `json:"quantity"`
	Amount        int       `json:"amount"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}

type Status struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type TransactionStatus struct {
	ID            int       `json:"id"`
	TransactionID int       `json:"transaction_id"`
	StatusID      int       `json:"status_id"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}

type Transaction struct {
	ID                  int       `json:"id"`
	TransactionStatusID int       `json:"transaction_status_id"`
	ExpMonth            int       `json:"exp_month"`
	ExpYear             int       `json:"exp_year"`
	Currency            string    `json:"currency"`
	LastFour            string    `json:"last_four"`
	BankReturnCode      string    `json:"bank_return_code"`
	PaymentIntent       string    `json:"payment_intent"`
	PaymentMethod       string    `json:"payment_method"`
	Amount              int       `json:"amount"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}

type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Customer struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (m *DBModel) GetWidget(id int) (Widget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var widget Widget
	row := m.DB.QueryRowContext(ctx, "select id, name, description, inventory_level, price, coalesce(image, ''), is_recurring, plan_id, created_at, updated_at from widgets where id = ?", id)

	err := row.Scan(&widget.ID, &widget.Name, &widget.Description, &widget.InventoryLevel, &widget.Price, &widget.Image, &widget.IsRecurring, &widget.PlanID, &widget.CreatedAt, &widget.UpdatedAt)

	if err != nil {
		return widget, err
	}

	return widget, nil
}

func (m *DBModel) CreateTransaction(txn Transaction) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	insert into transactions
	(amount, currency, last_four, bank_return_code, transaction_status_id, exp_month, exp_year, payment_intent, payment_method, created_at, updated_at)
	value (?,?,?,?,?,?,?,?,?,?,?)
`

	result, err := m.DB.ExecContext(ctx, query, txn.Amount, txn.Currency, txn.LastFour, txn.BankReturnCode,
		txn.TransactionStatusID, txn.ExpMonth, txn.ExpYear, txn.PaymentIntent, txn.PaymentMethod, time.Now(), time.Now())

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}
func (m *DBModel) CreateOrder(o Order) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	insert into orders
	(widget_id, transaction_id, customer_id, status_id, quantity, amount, created_at, updated_at)
	value (?,?,?,?,?,?,?,?)
`

	result, err := m.DB.ExecContext(ctx, query, o.WidgetID, o.TransactionID, o.CustomerID, o.StatusID, o.Quantity, o.Amount, time.Now(), time.Now())

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}
func (m *DBModel) CreateCustomer(c Customer) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	insert into customers
	(first_name, last_name, email, created_at, updated_at)
	value (?,?,?,?,?)
`

	result, err := m.DB.ExecContext(ctx, query, c.FirstName, c.LastName, c.Email, time.Now(), time.Now())

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *DBModel) CreateToken(token *Token, user User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `delete from tokens where user_id = ?`

	_, err := m.DB.ExecContext(ctx, query, user.ID)

	if err != nil {
		return err
	}

	query = `
	insert into tokens
	(user_id, name, email, token_hash, expires_on, created_at, updated_at)
	value (?,?,?,?,?,?,?)
`

	_, err = m.DB.ExecContext(ctx, query, user.ID, user.LastName, user.Email, token.Hash, token.Expiry, time.Now(), time.Now())

	if err != nil {
		return err
	}

	return nil
}

func (m *DBModel) GetUserByEmail(email string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email = strings.ToLower(email)
	var user User

	query := `select id, first_name, last_name, email, password, created_at, updated_at from users where email = ?`

	row := m.DB.QueryRowContext(ctx, query, email)
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return user, err
	}

	return user, nil
}

func (m *DBModel) GetUserByToken(token string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tokenHash := sha256.Sum256([]byte(token))
	var user User

	query := `
select u.id, u.first_name, u.last_name, u.email 
from users u inner join tokens t on (u.id = t.user_id) where t.token_hash = ? and t.expires_on > ?`

	err := m.DB.QueryRowContext(ctx, query, tokenHash[:], time.Now()).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &user, nil
}
func (m *DBModel) Authenticate(email, password string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	row := m.DB.QueryRowContext(ctx, "select id, password from users where email = ?", email)

	err := row.Scan(&id, &hashedPassword)

	if err != nil {
		return id, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return 0, errors.New("incorrect password")
	} else if err != nil {
		return 0, err
	}

	return id, nil
}
