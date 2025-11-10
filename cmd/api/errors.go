package main

import "net/http"

// errorResponse is a generic helper for sending JSON-formatted error messages
// to the client with a given status code.
func errorResponse(w http.ResponseWriter, status int, message any) {
	env := envelope{"error": message}
	if err := writeJSON(w, status, env, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
