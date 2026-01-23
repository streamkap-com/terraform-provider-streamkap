package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListTopics(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/details" {
			t.Errorf("Expected path /topics/details, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"page": 1, "page_size": 10, "total": 1, "has_next": false, "result": [{"id": "topic-1", "name": "test-topic"}]}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	topics, err := client.ListTopics(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}
	if len(topics.Result) != 1 {
		t.Errorf("Expected 1 topic, got %d", len(topics.Result))
	}
}

func TestListTopics_WithParams(t *testing.T) {
	// Setup mock server that verifies query parameters
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/details" {
			t.Errorf("Expected path /topics/details, got %s", r.URL.Path)
		}

		query := r.URL.RawQuery
		if !strings.Contains(query, "entity_type=sources") {
			t.Errorf("Expected entity_type=sources in query, got %s", query)
		}
		if !strings.Contains(query, "entity_id=source-1") {
			t.Errorf("Expected entity_id=source-1 in query, got %s", query)
		}
		if !strings.Contains(query, "entity_id=source-2") {
			t.Errorf("Expected entity_id=source-2 in query, got %s", query)
		}
		if !strings.Contains(query, "limit=50") {
			t.Errorf("Expected limit=50 in query, got %s", query)
		}
		if !strings.Contains(query, "offset=10") {
			t.Errorf("Expected offset=10 in query, got %s", query)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"page": 2, "page_size": 50, "total": 100, "has_next": true, "result": []}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	params := &TopicListParams{
		EntityType: "sources",
		EntityIDs:  []string{"source-1", "source-2"},
		Limit:      50,
		Offset:     10,
	}
	topics, err := client.ListTopics(context.Background(), params)
	if err != nil {
		t.Fatalf("ListTopics with params failed: %v", err)
	}
	if topics.PageSize != 50 {
		t.Errorf("Expected page_size 50, got %d", topics.PageSize)
	}
	if !topics.HasNext {
		t.Errorf("Expected has_next to be true")
	}
}

func TestListTopics_WithEntityDetails(t *testing.T) {
	// Setup mock server that returns full topic details including entity
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"page": 1,
			"page_size": 10,
			"total": 1,
			"has_next": false,
			"result": [{
				"id": "topic-123",
				"name": "my-topic",
				"prefix": "streamkap",
				"serialization": "avro",
				"messages_7d": 1000,
				"messages_30d": 5000,
				"entity": {
					"entity_type": "sources",
					"entity_id": "source-456",
					"name": "my-postgres",
					"connector": "postgresql",
					"display_name": "My PostgreSQL Source",
					"topic_ids": ["topic-123", "topic-124"],
					"topic_db_ids": ["db-1", "db-2"]
				}
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	topics, err := client.ListTopics(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}
	if len(topics.Result) != 1 {
		t.Fatalf("Expected 1 topic, got %d", len(topics.Result))
	}

	topic := topics.Result[0]
	if topic.ID != "topic-123" {
		t.Errorf("Expected topic ID 'topic-123', got '%s'", topic.ID)
	}
	if topic.Name != "my-topic" {
		t.Errorf("Expected topic name 'my-topic', got '%s'", topic.Name)
	}
	if topic.Entity == nil {
		t.Fatal("Expected entity to be present")
	}
	if topic.Entity.EntityType != "sources" {
		t.Errorf("Expected entity_type 'sources', got '%s'", topic.Entity.EntityType)
	}
	if topic.Entity.Connector != "postgresql" {
		t.Errorf("Expected connector 'postgresql', got '%s'", topic.Entity.Connector)
	}
	if len(topic.Entity.TopicIDs) != 2 {
		t.Errorf("Expected 2 topic_ids, got %d", len(topic.Entity.TopicIDs))
	}
}
