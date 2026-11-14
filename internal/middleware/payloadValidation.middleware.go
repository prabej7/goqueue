package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	apperrors "github.com/prabej7/goqueue/internal/utils/errors"
)

type contextKey string

const (
	BodyKey contextKey = "Body"
)

func ValidatePayload[T any](next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body T

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apperrors.ValidationError(w, "invalid payload", nil)
			return
		}

		if err := validator.New().Struct(body); err != nil {
			validationErrors, ok := err.(validator.ValidationErrors)

			if !ok {
				apperrors.ValidationError(w, "invalid json", nil)
				return
			}

			validationErrorMessages := buildValidationErrorMessage(validationErrors)
			apperrors.ValidationError(w, "inavlid payload", validationErrorMessages)
			return
		}

		ctx := context.WithValue(r.Context(), BodyKey, body)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getValidationErrorMessage(err validator.FieldError) string {
	var msg string

	switch err.Tag() {
	case "required":
		msg = fmt.Sprintf("%s is required", strings.ToLower(err.Field()))

	case "email":
		msg = fmt.Sprintf("%s must be a valid email", strings.ToLower(err.Field()))

	case "min":
		msg = fmt.Sprintf("%s must be at least %s characters", strings.ToLower(err.Field()), err.Param())
	default:
		msg = fmt.Sprintf("%s is invalid", strings.ToLower(err.Field()))
	}

	return msg
}

func buildValidationErrorMessage(validationErrors validator.ValidationErrors) map[string][]string {
	validationErrorsMap := make(map[string][]string)

	for _, err := range validationErrors {
		validationErrorsMap[strings.ToLower(err.Field())] = append(validationErrorsMap[strings.ToLower(err.Field())], getValidationErrorMessage(err))
	}

	return validationErrorsMap
}

func GetPayload[T any](r *http.Request) T {
	payload := r.Context().Value(BodyKey).(T)
	return payload
}

func GetParams[T any](r *http.Request) T {
	var params T

	decoder := schema.NewDecoder()
	routeParams := chi.RouteContext(r.Context()).URLParams
	values := make(map[string][]string, len(routeParams.Keys))
	
	for i, key := range routeParams.Keys {
		values[key] = []string{routeParams.Values[i]}
	}

	if err := decoder.Decode(&params, values); err != nil {
		return params
	}

	return params
}
