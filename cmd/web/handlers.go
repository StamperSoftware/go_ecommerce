package main

import (
	"ecommerce/internal/cards"
	"ecommerce/internal/encryption"
	"ecommerce/internal/models"
	"ecommerce/internal/urlsigner"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "terminal", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Home(w http.ResponseWriter, r *http.Request) {

	err := app.renderTemplate(w, r, "home", &templateData{})

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Login(w http.ResponseWriter, r *http.Request) {

	err := app.renderTemplate(w, r, "login", &templateData{})
	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PostLogin(w http.ResponseWriter, r *http.Request) {

	err := app.Session.RenewToken(r.Context())
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	err = r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	id, err := app.DB.Authenticate(email, password)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.Destroy(r.Context())
	_ = app.Session.RenewToken(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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

func (app *application) BronzePlan(w http.ResponseWriter, r *http.Request) {

	widget, err := app.DB.GetWidget(2)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})

	data["widget"] = widget

	err = app.renderTemplate(w, r, "bronze-plan", &templateData{
		Data: data,
	})

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) BronzePlanReceipt(w http.ResponseWriter, r *http.Request) {

	err := app.renderTemplate(w, r, "receipt-plan", &templateData{})

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ForgotPassword(w http.ResponseWriter, r *http.Request) {

	err := app.renderTemplate(w, r, "forgot-password", &templateData{})

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ResetPassword(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	url := r.RequestURI
	testURL := fmt.Sprintf("%s%s", app.config.frontend, url)

	signer := urlsigner.Signer{Secret: []byte(app.config.secretkey)}
	valid := signer.VerifyToken(testURL)

	if !valid {
		app.errorLog.Println("Invalid URL")
		return
	}

	isExpired := signer.IsTokenExpired(testURL, 60)

	if isExpired {
		app.errorLog.Println("Link expired")
		return
	}

	encryptor := encryption.Encryption{
		Key: []byte(app.config.secretkey),
	}

	encryptedEmail, err := encryptor.Encrypt(email)

	if err != nil {
		app.errorLog.Println("Encryption Failed", err)
		return
	}

	data := make(map[string]interface{})
	data["email"] = encryptedEmail

	err = app.renderTemplate(w, r, "reset-password", &templateData{Data: data})

	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "all-sales", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "all-subscriptions", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Sale(w http.ResponseWriter, r *http.Request) {

	stringMap := make(map[string]string)
	stringMap["title"] = "Sale"
	stringMap["cancel"] = "/admin/all-sales"
	stringMap["refund-url"] = "/admin/refund"
	stringMap["refund-text"] = "Refund"
	stringMap["refund-status-text"] = "Refunded"
	stringMap["refund-status-message"] = "Charge Refunded"

	if err := app.renderTemplate(w, r, "sale", &templateData{StringMap: stringMap}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Subscription(w http.ResponseWriter, r *http.Request) {

	stringMap := make(map[string]string)
	stringMap["title"] = "Subscription"
	stringMap["cancel"] = "/admin/all-subscriptions"
	stringMap["refund-url"] = "/admin/cancel-subscription"
	stringMap["refund-text"] = "Cancel Subscription"
	stringMap["refund-status-text"] = "Cancelled"
	stringMap["refund-status-message"] = "Subscription Cancelled"

	if err := app.renderTemplate(w, r, "sale", &templateData{StringMap: stringMap}); err != nil {
		app.errorLog.Println(err)
	}
}
