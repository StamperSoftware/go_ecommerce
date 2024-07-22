package main

import (
	"ecommerce/internal/cards"
	"ecommerce/internal/models"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "terminal", &templateData{}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}
func (app *application) Home(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

type TransactionData struct {
	FirstName       string
	LastName        string
	Email           string
	PaymentIntentID string
	PaymentMethodID string
	PaymentAmount   int
	PaymentCurrency string
	LastFour        string
	ExpMonth        int
	ExpYear         int
	BankReturnCode  string
}

func (app *application) GetTransactionData(r *http.Request) (TransactionData, error) {
	var txnData TransactionData

	err := r.ParseForm()

	if err != nil {
		app.errorLog.Println(err)
		return txnData, err
	}

	firstName := r.Form.Get("first-name")
	lastName := r.Form.Get("last-name")
	paymentIntent := r.Form.Get("payment-intent")
	paymentMethod := r.Form.Get("payment-method")
	paymentAmount := r.Form.Get("amount")
	paymentCurrency := r.Form.Get("payment-currency")
	email := r.Form.Get("email")

	amount, _ := strconv.Atoi(paymentAmount)

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.GetPaymentIntent(paymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		return txnData, err
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		return txnData, err
	}

	lastFour := pm.Card.Last4
	expMonth := pm.Card.ExpMonth
	expYear := pm.Card.ExpYear

	txnData = TransactionData{
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		PaymentIntentID: paymentIntent,
		PaymentMethodID: paymentMethod,
		PaymentAmount:   amount,
		PaymentCurrency: paymentCurrency,
		LastFour:        lastFour,
		ExpMonth:        int(expMonth),
		ExpYear:         int(expYear),
		BankReturnCode:  pi.LatestCharge.ID,
	}

	return txnData, nil
}

func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		app.errorLog.Println(err)
		return
	}
	widgetID, err := strconv.Atoi(r.Form.Get("product_id"))

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	txnData, err := app.GetTransactionData(r)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	customerID, err := app.SaveCustomer(txnData.FirstName, txnData.LastName, txnData.Email)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	txn := models.Transaction{
		TransactionStatusID: 2,
		ExpMonth:            txnData.ExpMonth,
		ExpYear:             txnData.ExpYear,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		BankReturnCode:      txnData.BankReturnCode,
		Amount:              txnData.PaymentAmount,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
	}

	transactionID, err := app.SaveTransaction(txn)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	order := models.Order{
		WidgetID:      widgetID,
		TransactionID: transactionID,
		CustomerID:    customerID,
		StatusID:      1,
		Quantity:      1,
		Amount:        txnData.PaymentAmount,
	}

	_, err = app.SaveOrder(order)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.Session.Put(r.Context(), "receipt", txnData)
	http.Redirect(w, r, "/receipt", http.StatusSeeOther)
}

func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {

	txnData, err := app.GetTransactionData(r)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	txn := models.Transaction{
		TransactionStatusID: 2,
		ExpMonth:            txnData.ExpMonth,
		ExpYear:             txnData.ExpYear,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		BankReturnCode:      txnData.BankReturnCode,
		Amount:              txnData.PaymentAmount,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
	}

	_, err = app.SaveTransaction(txn)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.Session.Put(r.Context(), "receipt", txnData)
	http.Redirect(w, r, "virtual-terminal-receipt", http.StatusSeeOther)
}

func (app *application) SaveCustomer(firstName, lastName, email string) (int, error) {
	customer := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	id, err := app.DB.CreateCustomer(customer)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (app *application) SaveTransaction(txn models.Transaction) (int, error) {

	id, err := app.DB.CreateTransaction(txn)

	if err != nil {
		return 0, err
	}

	return id, nil
}
func (app *application) SaveOrder(order models.Order) (int, error) {

	id, err := app.DB.CreateOrder(order)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (app *application) BuyOnce(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	widgetId, _ := strconv.Atoi(id)
	widget, err := app.DB.GetWidget(widgetId)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})
	data["widget"] = widget

	err = app.renderTemplate(w, r, "buy-once", &templateData{
		Data: data,
	}, "stripe-js")

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {

	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)
	app.Session.Remove(r.Context(), "receipt")
	data := make(map[string]interface{})
	data["txn"] = txn

	err := app.renderTemplate(w, r, "receipt", &templateData{
		Data: data,
	})

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) VirtualTerminalReceipt(w http.ResponseWriter, r *http.Request) {

	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)
	
	app.Session.Remove(r.Context(), "receipt")
	data := make(map[string]interface{})
	data["txn"] = txn

	err := app.renderTemplate(w, r, "virtual-terminal-receipt", &templateData{
		Data: data,
	})

	if err != nil {
		app.errorLog.Println(err)
	}
}
