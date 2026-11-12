package model

import "testing"

func TestJobSttaus(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected string
	}{
		{
			name:     "pending status",
			status:   JobStatusPending,
			expected: "pending",
		},
		{
			name:     "completed status",
			status:   JobStatusCompleted,
			expected: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}
