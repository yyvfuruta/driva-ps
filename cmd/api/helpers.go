package main

import (
	"encoding/json"
	"net/http"
)

type envelope map[string]any

// writeJSON helper takes the destination http.ResponseWriter, the HTTP status
// code to send, the data to encode to JSON, and a header map containing any
// additional HTTP headers we want to include in the response.
func writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(js)
	if err != nil {
		return err
	}
	return nil
}
