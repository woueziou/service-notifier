package service

import (
	"encoding/json"
	"testing"

	"github.com/flyasky/notifier/internal/model"
)

func TestMustMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name:  "string slice",
			input: []string{"a", "b", "c"},
			want:  `["a","b","c"]`,
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  `[]`,
		},
		{
			name:  "map",
			input: map[string]int{"key": 42},
			want:  `{"key":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustMarshalJSON(tt.input)
			if got != tt.want {
				t.Errorf("mustMarshalJSON(%v) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestMustMarshalJSON_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unmarshalable input")
		}
	}()

	mustMarshalJSON(make(chan int))
}

func TestModelJSONRoundtrip(t *testing.T) {
	job := model.Job{
		ID:         "j-123",
		ConsumerID: "c-456",
		Status:     model.JobStatusDelivered,
		To:         `["user@example.com"]`,
		Subject:    "Test",
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("failed to marshal job: %v", err)
	}

	var decoded model.Job
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal job: %v", err)
	}

	if decoded.ID != job.ID {
		t.Errorf("ID mismatch: %s != %s", decoded.ID, job.ID)
	}
	if decoded.Status != job.Status {
		t.Errorf("Status mismatch: %s != %s", decoded.Status, job.Status)
	}
}
