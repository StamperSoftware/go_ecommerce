package main

import (
	"bytes"
	"ecommerce/internal/cards"
	"ecommerce/internal/encryption"
	"ecommerce/internal/models"
	"ecommerce/internal/urlsigner"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v79"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"strings"
	"time"
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

type Invoice struct {
	ID        int       `json:"id"`
	Quantity  int       `json:"quantity"`
	Amount    int       `json:"amount"`
	Product   string    `json:"product"`
	CreatedAt time.Time `json:"created_at"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
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
			PaymentIntent:       sub.ID,
			PaymentMethod:       data.PaymentMethod,
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

		newOrderId, err := app.SaveOrder(order)

		if err != nil {
			app.errorLog.Println(err)
			return
		}

		inv := Invoice{
			ID:        newOrderId,
			Quantity:  order.Quantity,
			Amount:    order.Amount,
			Product:   "Bronze Plan Subscription",
			CreatedAt: time.Now(),
			FirstName: data.FirstName,
			LastName:  data.LastName,
			Email:     data.Email,
		}
		err = app.callInvoiceMicro(inv)

		if err != nil {
			app.errorLog.Println(err)
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

func (app *application) callInvoiceMicro(inv Invoice) error {
	url := "http://localhost:5000/invoice/create-and-send"

	out, err := json.MarshalIndent(inv, "", "\t")
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(out))

	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(request)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
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

func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	user, err := app.DB.GetUserByEmail(userInput.Email)

	if err != nil {
		_ = app.invalidCredentials(w)
		return
	}

	validPassword, err := app.doesPasswordMatch(user.Password, userInput.Password)

	if err != nil {
		_ = app.invalidCredentials(w)
		return
	}

	if !validPassword {
		_ = app.invalidCredentials(w)
		return
	}

	token, err := models.GenerateToken(user.ID, 24*time.Hour, models.ScopeAuthentication)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	err = app.DB.CreateToken(token, user)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var payload struct {
		Error   bool          `json:"error"`
		Message string        `json:"message"`
		Token   *models.Token `json:"authentication_token"`
	}

	payload.Error = false
	payload.Message = fmt.Sprintf("token for %s created", userInput.Email)
	payload.Token = token

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) authenticateToken(r *http.Request) (*models.User, error) {

	authorizationHeader := r.Header.Get("Authorization")

	if authorizationHeader == "" {
		return nil, errors.New("no authorization header")
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no authorization header")
	}

	token := headerParts[1]

	if len(token) != 26 {
		return nil, errors.New("token wrong size")
	}

	user, err := app.DB.GetUserByToken(token)

	if err != nil {
		return nil, errors.New("invalid user")
	}

	return user, nil
}

func (app *application) IsAuthenticated(w http.ResponseWriter, r *http.Request) {

	user, err := app.authenticateToken(r)

	if err != nil {
		_ = app.invalidCredentials(w)
		return
	}

	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	payload.Error = false
	payload.Message = fmt.Sprintf("authenticated user %s", user.Email)

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	var txnData struct {
		PaymentAmount   int    `json:"amount"`
		PaymentCurrency string `json:"currency"`
		FirstName       string `json:"first_name"`
		LastName        string `json:"last_name"`
		Email           string `json:"email"`
		PaymentIntent   string `json:"payment_intent"`
		PaymentMethod   string `json:"payment_method"`
		BankReturnCode  string `json:"bank_return_code"`
		ExpMonth        int    `json:"exp_month"`
		ExpYear         int    `json:"exp_year"`
		LastFour        string `json:"last_four"`
	}

	err := app.readJSON(w, r, &txnData)

	if err != nil {
		_ = app.badRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.GetPaymentIntent(txnData.PaymentIntent)

	if err != nil {
		_ = app.badRequest(w, r, err)
		return
	}
	pm, err := card.GetPaymentMethod(txnData.PaymentMethod)

	if err != nil {
		_ = app.badRequest(w, r, err)
		return
	}

	txnData.LastFour = pm.Card.Last4
	txnData.ExpMonth = int(pm.Card.ExpMonth)
	txnData.ExpYear = int(pm.Card.ExpYear)

	txn := models.Transaction{
		TransactionStatusID: 2,
		ExpMonth:            txnData.ExpMonth,
		ExpYear:             txnData.ExpYear,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		BankReturnCode:      pi.LatestCharge.ID,
		Amount:              txnData.PaymentAmount,
		PaymentMethod:       txnData.PaymentMethod,
		PaymentIntent:       txnData.PaymentIntent,
	}
	_, err = app.SaveTransaction(txn)

	if err != nil {
		_ = app.badRequest(w, r, err)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, txn)
}

func (app *application) ForgotPassword(w http.ResponseWriter, r *http.Request) {

	var payload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	link := fmt.Sprintf("%s/reset-password?email=%s", app.config.frontend, payload.Email)
	sign := urlsigner.Signer{
		Secret: []byte(app.config.secretkey),
	}

	_, err = app.DB.GetUserByEmail(payload.Email)

	if err != nil {
		var resp struct {
			Error   bool   `json:"error"`
			Message string `json:"message"`
		}

		resp.Error = true
		resp.Message = "No matching Email Found"

		app.writeJSON(w, http.StatusAccepted, resp)
		return
	}

	signedLink := sign.GenerateTokenFromString(link)

	var data struct {
		Link string
	}

	data.Link = signedLink
	err = app.SendMail("info@widgets.com", payload.Email, "Password Reset Request", "password-reset", data)

	if err != nil {
		app.errorLog.Println(err)
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false

	app.writeJSON(w, http.StatusCreated, resp)

}

func (app *application) ResetPassword(w http.ResponseWriter, r *http.Request) {

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	encryptor := encryption.Encryption{Key: []byte(app.config.secretkey)}
	email, err := encryptor.Decrypt(payload.Email)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	user, err := app.DB.GetUserByEmail(email)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	err = app.DB.UpdatePasswordForUser(user, string(newHash))

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Message = "Password Updated"

	app.writeJSON(w, http.StatusCreated, resp)

}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {

	var payload struct {
		PageSize    int `json:"page_size"`
		CurrentPage int `json:"page"`
	}

	err := app.readJSON(w, r, &payload)

	allSales, lastPage, totalRecords, err := app.DB.GetAllOrders(payload.PageSize, payload.CurrentPage)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	var resp struct {
		CurrentPage  int             `json:"current_page"`
		PageSize     int             `json:"page_size"`
		LastPage     int             `json:"last_page"`
		TotalRecords int             `json:"total_records"`
		Orders       []*models.Order `json:"orders"`
	}

	resp.CurrentPage = payload.CurrentPage
	resp.PageSize = payload.PageSize
	resp.LastPage = lastPage
	resp.TotalRecords = totalRecords
	resp.Orders = allSales

	app.writeJSON(w, http.StatusOK, resp)

}

func (app *application) Sale(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	orderID, _ := strconv.Atoi(id)

	order, err := app.DB.GetOrderById(orderID)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, order)
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {

	var payload struct {
		PageSize    int `json:"page_size"`
		CurrentPage int `json:"page"`
	}

	err := app.readJSON(w, r, &payload)
	
	allSubscriptions, lastPage, totalRecords, err := app.DB.GetAllSubscriptions(payload.PageSize, payload.CurrentPage)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		CurrentPage  int             `json:"current_page"`
		PageSize     int             `json:"page_size"`
		LastPage     int             `json:"last_page"`
		TotalRecords int             `json:"total_records"`
		Orders       []*models.Order `json:"orders"`
	}

	resp.CurrentPage = payload.CurrentPage
	resp.PageSize = payload.PageSize
	resp.LastPage = lastPage
	resp.TotalRecords = totalRecords
	resp.Orders = allSubscriptions

	app.writeJSON(w, http.StatusOK, resp)

}

func (app *application) Refund(w http.ResponseWriter, r *http.Request) {
	var chargeToRefund struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"pi"`
		Amount        int    `json:"amount"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &chargeToRefund)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: chargeToRefund.Currency,
	}

	err = card.Refund(chargeToRefund.PaymentIntent, chargeToRefund.Amount)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	err = app.DB.UpdateOrderStatus(chargeToRefund.ID, 2)

	if err != nil {
		app.badRequest(w, r, errors.New("charge was refunded but the database could not be updated"))
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Message = "Charge Refunded"

	app.writeJSON(w, http.StatusOK, resp)

}

func (app *application) CancelSubscription(w http.ResponseWriter, r *http.Request) {

	var subToCancel struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"pi"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &subToCancel)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: subToCancel.Currency,
	}

	err = card.CancelSubscription(subToCancel.PaymentIntent)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	err = app.DB.UpdateOrderStatus(subToCancel.ID, 3)

	if err != nil {
		app.badRequest(w, r, errors.New("subscription was cancelled but the database could not be updated"))
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Message = "Subscription Cancelled"

	app.writeJSON(w, http.StatusOK, resp)

}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	allUsers, err := app.DB.GetAllUsers()

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, allUsers)
}
func (app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	userID, _ := strconv.Atoi(id)
	err := app.DB.DeleteUser(userID)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "User was Deleted"
	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) User(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, _ := strconv.Atoi(id)

	user, err := app.DB.GetUserById(userID)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, user)
}

func (app *application) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, _ := strconv.Atoi(id)
	var user models.User
	err := app.readJSON(w, r, &user)

	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	if userID > 0 {
		err = app.DB.EditUser(user)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

		if user.Password != "" {
			newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
			if err != nil {
				app.badRequest(w, r, err)
				return
			}

			err = app.DB.UpdatePasswordForUser(user, string(newHash))
			if err != nil {
				app.badRequest(w, r, err)
				return
			}

		}
	} else {
		newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

		err = app.DB.CreateUser(user, string(newHash))
		if err != nil {
			app.badRequest(w, r, err)
			return
		}

	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "User Updated"

	app.writeJSON(w, http.StatusOK, user)
}
