package main

import (
	"ecommerce/internal/cards"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "terminal", &templateData{}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	cardHolder := r.Form.Get("cardholder-name")
	paymentIntent := r.Form.Get("payment-intent")
	paymentMethod := r.Form.Get("payment-method")
	paymentAmount := r.Form.Get("amount")
	paymentCurrency := r.Form.Get("payment-currency")
	email := r.Form.Get("email")

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.GetPaymentIntent(paymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	lastFour := pm.Card.Last4
	expirationMonth := pm.Card.ExpMonth
	expirationYear := pm.Card.ExpYear

	data := make(map[string]interface{})

	data["cardholder"] = cardHolder
	data["email"] = email
	data["payment-intent"] = paymentIntent
	data["payment-method"] = paymentMethod
	data["payment-amount"] = paymentAmount
	data["payment-currency"] = paymentCurrency
	data["last-four"] = lastFour
	data["expiration-month"] = expirationMonth
	data["expiration-year"] = expirationYear
	data["bank-return-code"] = pi.LatestCharge.ID

	if err = app.renderTemplate(w, r, "succeeded", &templateData{Data: data}); err != nil {
		app.errorLog.Println(err)
	}
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
