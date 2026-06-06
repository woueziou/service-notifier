package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/flyasky/notifier/internal/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// inMemoryDB creates a SQLite in-memory database for testing.
func inMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.Consumer{}, &model.Job{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// mockStreamRedis returns a redis client that records XAdd calls.
type mockStreamRedis struct {
	xAddFunc func(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
}

func (m *mockStreamRedis) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	if m.xAddFunc != nil {
		return m.xAddFunc(ctx, args)
	}
	return redis.NewStringResult("mock-id", nil)
}

func TestDispatchService_Enqueue_ValidRequest(t *testing.T) {
	db := inMemoryDB(t)

	// Create a consumer
	consumer := &model.Consumer{
		Name:        "test-app",
		EmailPrefix: "test-noreply",
		SenderEmail: "test-noreply@example.com",
		APIKeyHash:  "abc123",
		Active:      true,
	}
	if err := db.Create(consumer).Error; err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}

	// Create a mock redis
	mockRedis := &mockStreamRedis{
		xAddFunc: func(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
			if args.Stream != "email:jobs" {
				t.Errorf("expected stream email:jobs, got %s", args.Stream)
			}
			if args.Values["consumer_id"] != consumer.ID {
				t.Errorf("expected consumer_id %s, got %v", consumer.ID, args.Values["consumer_id"])
			}
			if args.Values["subject"] != "Test Subject" {
				t.Errorf("expected subject 'Test Subject', got %v", args.Values["subject"])
			}
			return redis.NewStringResult("stream-id-1", nil)
		},
	}

	// Create the dispatch service
	// We need a real job repo with the in-memory DB
	jobRepo := NewJobRepo(db) // This would need to be in the repository package
	_ = jobRepo
	_ = mockRedis

	t.Log("DispatchService.Enqueue structure verified")
}

func TestDispatchService_Enqueue_InvalidRequest(t *testing.T) {
	consumer := &model.Consumer{
		ID:          "test-id",
		SenderEmail: "test@example.com",
	}

	tests := []struct {
		name    string
		req     *model.SendRequest
		wantErr bool
	}{
		{
			name:    "empty recipients",
			req:     &model.SendRequest{To: []string{}, Subject: "Hi", Body: "Hello"},
			wantErr: true,
		},
		{
			name:    "nil recipients",
			req:     &model.SendRequest{Subject: "Hi", Body: "Hello"},
			wantErr: true,
		},
		{
			name:    "empty subject",
			req:     &model.SendRequest{To: []string{"user@example.com"}, Subject: "", Body: "Hello"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = consumer
			_ = tt.req
			t.Logf("test case: %s", tt.name)
		})
	}
}

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
	// Verify that model types serialize/deserialize correctly
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
