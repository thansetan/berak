package helper

import (
	"encoding/json"
	"net/http"
)

func OurFault(w http.ResponseWriter) {
	WriteMessage(w, http.StatusInternalServerError, "it's our fault, not yours!")
}

func WriteMessage(w http.ResponseWriter, statusCode int, msg string) {
	WriteResponseJSON(w, statusCode, struct {
		Message string `json:"message"`
	}{
		Message: msg,
	})
}

func WriteResponseJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
