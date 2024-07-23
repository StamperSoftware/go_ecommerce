package main

import (
	"ecommerce/internal/cards"
	"ecommerce/internal/models"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v79"
	"net/http"
	"strconv"
)

type stripePayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	LastFour      string `json:"last_four"`
	Plan          string `json:"plan"`
	CardBrand     string `json:"card_brand"`
	ExpMonth      int    `json:"exp_month"`
	ExpYear       int    `json:"exp_year"`
	ProductID     string `json:"product_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

type jsonResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      int    `json:"id,omitempty"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {

	var payload stripePayload

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		app.errorLog.Println(err)
		return
	}
	amount, err := strconv.Atoi(payload.Amount)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	okay := true

	pi, msg, err := card.Charge(payload.Currency, amount)

	if err != nil {
		okay = false
	}

	if okay {
		out, err := json.MarshalIndent(pi, "", "    ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	} else {

		j := jsonResponse{
			OK:      false,
			Message: msg,
			Content: "",
		}
		out, err := json.MarshalIndent(j, "", "   ")

		if err != nil {
			app.errorLog.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}

}

func (app *application) GetWidgetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	widgetId, _ := strconv.Atoi(id)
	widget, err := app.DB.GetWidget(widgetId)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	out, err := json.MarshalIndent(widget, "", "    ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)

}

func (app *application) CreateCustomerAndSubscribe(w http.ResponseWriter, r *http.Request) {

	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: data.Currency,
	}

	okay := true
	var sub *stripe.Subscription
	txnMessage := "Transaction Successful"
	stripeCustomer, msg, err := card.CreateCustomer(data.PaymentMethod, data.Email)

	if err != nil {
		app.errorLog.Println(err)
		okay = false
		txnMessage = msg
	}

	if okay {
		sub, err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email, data.LastFour, "")

		if err != nil {
			app.errorLog.Println(err)
			okay = false
			txnMessage = "Error Subscribing Customer"
		}

		app.infoLog.Println("Subscription id, ", sub.ID)
	}

	if okay {
		productID, _ := strconv.Atoi(data.ProductID)
		customerID, err := app.SaveCustomer(data.FirstName, data.LastName, data.Email)

		if err != nil {
			app.errorLog.Println(err)
			return
		}

		amount, _ := strconv.Atoi(data.Amount)

		txn := models.Transaction{
			ExpMonth:            data.ExpMonth,
			ExpYear:             data.ExpYear,
			Currency:            "usd",
			LastFour:            data.LastFour,
			TransactionStatusID: 2,
			Amount:              amount,
		}

		txnID, err := app.SaveTransaction(txn)

		if err != nil {
			app.errorLog.Println(err)
			return
		}

		order := models.Order{
			WidgetID:      productID,
			TransactionID: txnID,
			CustomerID:    customerID,
			StatusID:      1,
			Quantity:      1,
			Amount:        amount,
		}

		_, err = app.SaveOrder(order)

		if err != nil {
			app.errorLog.Println(err)
			return
		}

	}

	resp := jsonResponse{
		OK:      okay,
		Message: txnMessage,
	}

	out, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)

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
