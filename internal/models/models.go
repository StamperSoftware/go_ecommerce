package models

import (
	"context"
	"database/sql"
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
	row := m.DB.QueryRowContext(ctx, "select id, name, description, inventory_level, price, coalesce(image, ''), created_at, updated_at from widgets where id = ?", id)

	err := row.Scan(&widget.ID, &widget.Name, &widget.Description, &widget.InventoryLevel, &widget.Price, &widget.Image, &widget.CreatedAt, &widget.UpdatedAt)

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
