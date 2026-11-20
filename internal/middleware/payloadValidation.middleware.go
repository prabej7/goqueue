package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/schema"
	apperrors "github.com/prabej7/goqueue/internal/utils/errors"
)

type contextKey string

const (
	BodyKey  contextKey = "Body"
	QueryKey contextKey = "Query"
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

func ValidateQuery[T any](next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var query T

		if err := schema.NewDecoder().Decode(&query, r.URL.Query()); err != nil {
			apperrors.ValidationError(w, "invalid query", nil)
			return
		}

		if err := validator.New().Struct(&query); err != nil {
			validationErrors, ok := err.(validator.ValidationErrors)
			if !ok {
				apperrors.ValidationError(w, "invalid query", nil)
				return
			}

			apperrors.ValidationError(
				w,
				"invalid query",
				buildValidationErrorMessage(validationErrors),
			)
			return
		}

		ctx := context.WithValue(r.Context(), QueryKey, &query)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetQuery[T any](r *http.Request) T {
	return *r.Context().Value(QueryKey).(*T)
}

func ValidateUUID(id string) error {
	if len(id) != 36 {
		return fmt.Errorf("invalid UUID length")
	}

	_, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format")
	}

	return nil
}

func ValidateUUIDParam(param string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")

			err := ValidateUUID(id)
			if err != nil {
				apperrors.ValidationError(
					w,
					"invalid id",
					nil,
				)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
