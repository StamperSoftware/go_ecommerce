package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(SessionLoad)
	mux.Get("/", app.Home)
	mux.Get("/ws", app.WsEndpoint)
	mux.Get("/logout", app.Logout)
	mux.Get("/login", app.Login)
	mux.Post("/login", app.PostLogin)

	mux.Route("/admin", func(mux chi.Router) {
		mux.Use(app.Auth)
		mux.Get("/virtual-terminal", app.VirtualTerminal)
		mux.Get("/all-users", app.AllUsers)
		mux.Get("/users/{id}", app.User)

		mux.Get("/all-sales", app.AllSales)
		mux.Get("/sales/{id}", app.Sale)

		mux.Get("/all-subscriptions", app.AllSubscriptions)
		mux.Get("/subscriptions/{id}", app.Subscription)

	})

	mux.Get("/receipt", app.Receipt)
	mux.Get("/plans/bronze", app.BronzePlan)
	mux.Get("/receipt/bronze", app.BronzePlanReceipt)
	mux.Post("/payment-succeeded", app.PaymentSucceeded)
	mux.Get("/forgot-password", app.ForgotPassword)
	mux.Get("/reset-password", app.ResetPassword)

	mux.Get("/widget/{id}", app.BuyOnce)

	fileServer := http.FileServer(http.Dir("./static"))

	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}
