package api

import (
	"context"
	"encoding/json"
	"fmt"
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

// TestListTopics_WithParams pins the two wire-format facts the backend cares
// about: page/page_size (there is no limit/offset — TopicDetailsReq never
// declared them) and entity_id as ONE comma-separated value. Sending entity_id
// as a repeated key collapsed a multi-ID filter to a single arbitrary ID,
// because the backend field is a scalar it splits on commas.
func TestListTopics_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/details" {
			t.Errorf("Expected path /topics/details, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if got := query.Get("entity_type"); got != "sources" {
			t.Errorf("Expected entity_type=sources, got %q", got)
		}
		if got := query["entity_id"]; len(got) != 1 {
			t.Errorf("Expected exactly one entity_id param, got %v", got)
		}
		if got := query.Get("entity_id"); got != "source-1,source-2" {
			t.Errorf("Expected comma-joined entity_id, got %q", got)
		}
		if got := query.Get("page"); got != "1" {
			t.Errorf("Expected page=1, got %q", got)
		}
		if got := query.Get("page_size"); got != "100" {
			t.Errorf("Expected page_size=100, got %q", got)
		}
		if query.Has("limit") || query.Has("offset") {
			t.Errorf("limit/offset are not backend params and must not be sent, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"page": 1, "page_size": 100, "total": 1, "has_next": false, "result": [{"id": "topic-1", "name": "test-topic"}]}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	params := &TopicListParams{
		EntityType: "sources",
		EntityIDs:  []string{"source-1", "source-2"},
	}
	topics, err := client.ListTopics(context.Background(), params)
	if err != nil {
		t.Fatalf("ListTopics with params failed: %v", err)
	}
	if len(topics.Result) != 1 {
		t.Errorf("Expected 1 topic, got %d", len(topics.Result))
	}
}

// TestListTopics_Paginates proves the data source no longer truncates at the
// backend's default page of 10: ListTopics must keep paging until a short page.
func TestListTopics_Paginates(t *testing.T) {
	const fullPage = 100
	const lastPage = 5

	pagesServed := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pagesServed = append(pagesServed, page)

		count := fullPage
		if page == "2" {
			count = lastPage
		}
		topics := make([]string, 0, count)
		for i := 0; i < count; i++ {
			topics = append(topics, fmt.Sprintf(`{"id": "topic-%s-%d", "name": "t-%s-%d"}`, page, i, page, i))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"page": %s, "page_size": 100, "total": %d, "has_next": %t, "result": [%s]}`,
			page, fullPage+lastPage, page == "1", strings.Join(topics, ","))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	topics, err := client.ListTopics(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}
	if len(topics.Result) != fullPage+lastPage {
		t.Errorf("Expected %d topics across both pages, got %d", fullPage+lastPage, len(topics.Result))
	}
	if topics.Total != fullPage+lastPage {
		t.Errorf("Expected total %d, got %d", fullPage+lastPage, topics.Total)
	}
	if len(pagesServed) != 2 || pagesServed[0] != "1" || pagesServed[1] != "2" {
		t.Errorf("Expected pages 1 and 2 to be requested, got %v", pagesServed)
	}
}

// TestListTopics_LongEntityIDListUsesSearch — past the URL-length threshold the
// entity IDs move into the body of POST /topics/details/search, the backend's
// documented alternative for lists that would blow up the query string.
func TestListTopics_LongEntityIDListUsesSearch(t *testing.T) {
	entityIDs := make([]string, 200)
	for i := range entityIDs {
		entityIDs[i] = fmt.Sprintf("source_%024d", i)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/details/search" {
			t.Errorf("Expected path /topics/details/search, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Query().Has("entity_id") {
			t.Errorf("entity_id must move to the body, but was still on the query string: %s", r.URL.RawQuery)
		}
		if got := r.URL.Query().Get("page_size"); got != "100" {
			t.Errorf("Expected page_size=100 on the search request, got %q", got)
		}

		var body topicsSearchBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode search body: %v", err)
		}
		if len(body.EntityID) != len(entityIDs) {
			t.Errorf("Expected %d entity IDs in the body, got %d", len(entityIDs), len(body.EntityID))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"page": 1, "page_size": 100, "total": 1, "has_next": false, "result": [{"id": "topic-1", "name": "test-topic"}]}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	topics, err := client.ListTopics(context.Background(), &TopicListParams{EntityIDs: entityIDs})
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}
	if len(topics.Result) != 1 {
		t.Errorf("Expected 1 topic, got %d", len(topics.Result))
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
				"serialization": {
					"key_format": "string",
					"value_format": "avro",
					"key_converter": "org.apache.kafka.connect.storage.StringConverter",
					"value_converter": "io.confluent.connect.avro.AvroConverter",
					"schema_registry_enabled": true
				},
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
	if topic.Serialization == nil {
		t.Fatal("Expected serialization object to be present")
	}
	if topic.Serialization.ValueFormat != "avro" {
		t.Errorf("Expected value_format 'avro', got '%s'", topic.Serialization.ValueFormat)
	}
	if topic.Serialization.KeyFormat != "string" {
		t.Errorf("Expected key_format 'string', got '%s'", topic.Serialization.KeyFormat)
	}
	if !topic.Serialization.SchemaRegistryEnabled {
		t.Error("Expected schema_registry_enabled to be true")
	}
}

func TestGetTopicTableMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/table_metrics" {
			t.Errorf("Expected path /topics/table_metrics, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"entity-1": {"topic-1": {"messages_in": 100}}}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	metrics, err := client.GetTopicTableMetrics(context.Background(), TopicTableMetricsRequest{
		Entities: []TopicMetricsEntity{{ID: "entity-1", EntityType: "sources", Connector: "postgresql", TopicIDs: []string{"topic-1"}, TopicDBIDs: []string{}}},
	})
	if err != nil {
		t.Fatalf("GetTopicTableMetrics failed: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics, got nil")
	}
}

func TestGetTopicDetailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/topic-123" {
			t.Errorf("Expected path /topics/topic-123, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "detailed=true" {
			t.Errorf("Expected query detailed=true, got %s", r.URL.RawQuery)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		// Shape mirrors the backend's TopicDetailsWithKafka: kafka.partitions is
		// an object (TopicKafkaPartitions), and kafka.configs keys are
		// underscored strings (TopicKafkaConfigs), not dotted Kafka property
		// names carrying numbers.
		w.Write([]byte(`{
			"id": "topic-123",
			"name": "my-topic",
			"prefix": "streamkap",
			"serialization": {
				"key_format": "string",
				"value_format": "avro",
				"schema_registry_enabled": true
			},
			"entity": {
				"entity_type": "sources",
				"entity_id": "source-456",
				"name": "my-postgres"
			},
			"kafka": {
				"partitions": {
					"count": 3,
					"replication_factor": 3,
					"under_replicated_count": 0,
					"offline_count": 0,
					"details": []
				},
				"configs": {
					"retention_ms": "604800000",
					"cleanup_policy": "delete"
				},
				"health": {
					"is_healthy": true,
					"under_replicated_partitions": 0,
					"offline_partitions": 0,
					"total_replicas": 3,
					"in_sync_replicas": 3
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	topic, err := client.GetTopicDetailed(context.Background(), "topic-123")
	if err != nil {
		t.Fatalf("GetTopicDetailed failed: %v", err)
	}
	if topic.ID != "topic-123" {
		t.Errorf("Expected ID topic-123, got %s", topic.ID)
	}
	if topic.Name != "my-topic" {
		t.Errorf("Expected name my-topic, got %s", topic.Name)
	}
	if topic.Kafka == nil || topic.Kafka.Partitions.Count != 3 {
		t.Error("Expected 3 partitions")
	}
	if topic.Kafka.Configs == nil || topic.Kafka.Configs.RetentionMs == nil || *topic.Kafka.Configs.RetentionMs != "604800000" {
		t.Error("Expected retention_ms 604800000")
	}
	if topic.Kafka.Configs.CleanupPolicy == nil || *topic.Kafka.Configs.CleanupPolicy != "delete" {
		t.Error("Expected cleanup_policy delete")
	}
	if topic.Entity == nil || topic.Entity.EntityID != "source-456" {
		t.Error("Expected entity_id source-456")
	}
	if topic.Serialization == nil || topic.Serialization.ValueFormat != "avro" {
		t.Error("Expected serialization value_format avro")
	}
}

func TestGetTopicDetailed_MinimalResponse(t *testing.T) {
	// Test handling of minimal response without optional fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "topic-minimal",
			"name": "minimal-topic"
		}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})

	topic, err := client.GetTopicDetailed(context.Background(), "topic-minimal")
	if err != nil {
		t.Fatalf("GetTopicDetailed failed: %v", err)
	}
	if topic.ID != "topic-minimal" {
		t.Errorf("Expected ID topic-minimal, got %s", topic.ID)
	}
	if topic.Entity != nil {
		t.Error("Expected entity to be nil")
	}
	if topic.Kafka != nil {
		t.Error("Expected kafka to be nil")
	}
	if topic.Prefix != nil {
		t.Error("Expected prefix to be nil")
	}
	if topic.Serialization != nil {
		t.Error("Expected serialization to be nil for minimal response")
	}
}
