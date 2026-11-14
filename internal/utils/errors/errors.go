package apperrors

import (
	"log"
	"net/http"

	"github.com/prabej7/goqueue/internal/utils/response"
)

type Error struct {
	StatusCode int
	Message    string
	Err        error
}

func WriteError(w http.ResponseWriter, err *Error) error {
	return response.SendResponse(w, err.StatusCode, nil)
}

func ValidationError(w http.ResponseWriter, message string, validationErrors map[string][]string) error {
	return response.SendResponse(w, http.StatusBadRequest, &response.Response{
		Message: message,
		Errors:  validationErrors,
	})
}

func InternalServerError(w http.ResponseWriter, err error) error {
	log.Printf("internal server error: %+v", err)
	return response.SendResponse(w, http.StatusInternalServerError, &response.Response{
		Message: "internal server error",
	})
}

func NotFoundError(w http.ResponseWriter, message string) error {
	return response.SendResponse(w, http.StatusNotFound, &response.Response{
		Message: message,
	})
}
