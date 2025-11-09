package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /orders", authMiddleware(app.createOrderHandler))
	mux.HandleFunc("GET /orders/{id}", app.getOrderHandler)
	mux.HandleFunc("GET /healthz", app.healthzHandler)
	mux.HandleFunc("GET /readyz", app.readyzHandler)

	return mux
}
