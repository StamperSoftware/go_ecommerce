package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	mux.Post("/api/payment-intent", app.GetPaymentIntent)
	mux.Get("/api/widget/{id}", app.GetWidgetByID)
	mux.Post("/api/forgot-password", app.ForgotPassword)
	mux.Post("/api/reset-password", app.ResetPassword)
	mux.Post("/api/create-customer-and-subscribe", app.CreateCustomerAndSubscribe)
	mux.Post("/api/authenticate", app.CreateAuthToken)
	mux.Post("/api/is-authenticated", app.IsAuthenticated)
	mux.Route("/api/admin", func(mux chi.Router) {
		mux.Use(app.Auth)
		mux.Post("/virtual-terminal-succeeded", app.VirtualTerminalPaymentSucceeded)
		mux.Post("/all-sales", app.AllSales)
		mux.Post("/sales/{id}", app.Sale)
		mux.Post("/all-subscriptions", app.AllSubscriptions)
		mux.Post("/refund", app.Refund)
		mux.Post("/cancel-subscription", app.CancelSubscription)

		mux.Post("/all-users", app.AllUsers)
		mux.Post("/users/{id}", app.User)
		mux.Put("/users/{id}", app.UpdateUser)
		mux.Delete("/users/{id}", app.DeleteUser)
	})

	return mux
}
