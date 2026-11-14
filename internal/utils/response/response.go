package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

func SendResponse(w http.ResponseWriter, statusCode int, response *Response) error {
	w.Header().Set("Content-Type", "text/json")
	w.WriteHeader(statusCode)

	return json.NewEncoder(w).Encode(response)
}
