package tests

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func AssertHttpResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int) {
	if rec.Code != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, rec.Code)
	}
}

func GetJsonPayload(payload any) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	stringJsonBody := string(jsonPayload)

	return stringJsonBody, nil
}
