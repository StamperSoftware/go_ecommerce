package main

import "net/http"

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)

	stringMap["publishable_key"] = app.config.stripe.key

	if err := app.renderTemplate(w, r, "terminal", &templateData{StringMap: stringMap}); err != nil {
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

	data := make(map[string]interface{})

	data["cardholder"] = cardHolder
	data["email"] = email
	data["payment-intent"] = paymentIntent
	data["payment-method"] = paymentMethod
	data["payment-amount"] = paymentAmount
	data["payment-currency"] = paymentCurrency

	if err = app.renderTemplate(w, r, "succeeded", &templateData{Data: data}); err != nil {
		app.errorLog.Println(err)
	}
}
