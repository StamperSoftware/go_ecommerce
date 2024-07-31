﻿package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(SessionLoad)
	mux.Get("/", app.Home)
	mux.Get("/logout", app.Logout)
	mux.Get("/login", app.Login)
	mux.Post("/login", app.PostLogin)

	mux.Route("/admin", func(mux chi.Router) {
		mux.Use(app.Auth)
		mux.Get("/virtual-terminal", app.VirtualTerminal)

	})

	mux.Post("/virtual-terminal-payment-succeeded", app.VirtualTerminalPaymentSucceeded)
	mux.Get("/receipt", app.Receipt)
	mux.Get("/virtual-terminal-receipt", app.VirtualTerminalReceipt)
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
