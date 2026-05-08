package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Topic mirrors the relevant slice of the backend's GET /topics/{id} response
// (TopicDetailsWithKafka). UpdateTopic does NOT marshal this struct directly —
// it builds the `{"payload": {...}}` envelope by hand to match the backend's
// at-least-one-of partition_count/tags rule on UpdateTopicReqBody.
//
// Tags has no omitempty: backend distinguishes `null`/absent ("keep existing")
// from `[]` ("clear"). See api.Source.Tags for the long version.
type Topic struct {
	// TopicID is populated from the response field `id`. The backend's
	// TopicDetailsRes carries no `topic_id` key — only `id` — so the previous
	// `json:"topic_id"` tag silently failed to deserialize on Read. Field name
	// kept as TopicID for callers; only the JSON tag changed.
	TopicID string `json:"id"`
	// PartitionCount is filled from `kafka.partitions.count` after Unmarshal —
	// the backend does not return a top-level `partition_count` key, so we copy
	// it out of the nested struct explicitly in GetTopic.
	PartitionCount int             `json:"-"`
	Tags           []string        `json:"tags"`
	Kafka          *TopicKafkaInfo `json:"kafka,omitempty"`
}

// TopicKafkaInfo captures only the field of TopicKafkaMetadata we care about
// (live partition count) — the full backend struct is much larger.
type TopicKafkaInfo struct {
	Partitions TopicPartitionsInfo `json:"partitions"`
}

// TopicPartitionsInfo carries the live partition count from the broker.
type TopicPartitionsInfo struct {
	Count int `json:"count"`
}

// TopicEntity represents the entity (source/transform/destination) that owns a topic
type TopicEntity struct {
	EntityType  string   `json:"entity_type"`  // "sources", "transforms", "destinations"
	EntityID    string   `json:"entity_id"`
	Name        string   `json:"name"`
	Connector   string   `json:"connector"`
	DisplayName string   `json:"display_name"`
	TopicIDs    []string `json:"topic_ids"`
	TopicDBIDs  []string `json:"topic_db_ids"`
}

// TopicDetails represents detailed topic information from /topics/details
type TopicDetails struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Entity        *TopicEntity `json:"entity,omitempty"`
	Prefix        *string      `json:"prefix,omitempty"`
	Serialization *string      `json:"serialization,omitempty"`
	Messages7D    *int64       `json:"messages_7d,omitempty"`
	Messages30D   *int64       `json:"messages_30d,omitempty"`
}

// TopicDetailsResponse represents the paginated response from /topics/details
type TopicDetailsResponse struct {
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int            `json:"total"`
	HasNext  bool           `json:"has_next"`
	Result   []TopicDetails `json:"result"`
}

// TopicListParams represents query parameters for listing topics
type TopicListParams struct {
	EntityType string   // Filter by entity type: "sources", "transforms", "destinations"
	EntityIDs  []string // Filter by specific entity IDs
	Limit      int      // Pagination limit
	Offset     int      // Pagination offset
}

// TopicMetricsEntity represents an entity with its topic IDs for metrics request
// Must match backend SingleTopicTableMetricReq model
type TopicMetricsEntity struct {
	ID         string   `json:"id"`           // Entity ID
	EntityType string   `json:"entity_type"`  // "sources", "transforms", "destinations"
	Connector  string   `json:"connector"`    // Connector type
	TopicIDs   []string `json:"topic_ids"`    // List of topic IDs
	TopicDBIDs []string `json:"topic_db_ids"` // List of topic DB IDs (MongoDB ObjectIds)
}

// TopicTableMetricsRequest represents the request body for /topics/table_metrics
type TopicTableMetricsRequest struct {
	Entities      []TopicMetricsEntity `json:"entities"`
	TimestampFrom *string              `json:"timestamp_from,omitempty"` // ISO 8601 string
	TimestampTo   *string              `json:"timestamp_to,omitempty"`   // ISO 8601 string
	TimeType      *string              `json:"time_type,omitempty"`      // "latest", "timeseries", "timesummary"
	TimeInterval  *int                 `json:"time_interval,omitempty"`
	TimeUnit      *string              `json:"time_unit,omitempty"` // "minute", "hour", "day", "week", "month"
}

// TopicMetrics represents metrics for a single topic
type TopicMetrics struct {
	MessagesIn   *int64   `json:"messages_in,omitempty"`
	MessagesOut  *int64   `json:"messages_out,omitempty"`
	BytesIn      *int64   `json:"bytes_in,omitempty"`
	BytesOut     *int64   `json:"bytes_out,omitempty"`
	Lag          *int64   `json:"lag,omitempty"`
	AvgLatencyMs *float64 `json:"avg_latency_ms,omitempty"`
}

// TopicTableMetricsResponse represents the response from /topics/table_metrics
// Map structure: entity_id -> topic_id -> metrics
type TopicTableMetricsResponse map[string]map[string]TopicMetrics

// TopicKafkaConfig represents Kafka configuration for a topic
type TopicKafkaConfig struct {
	RetentionMs   *int64  `json:"retention.ms,omitempty"`
	CleanupPolicy *string `json:"cleanup.policy,omitempty"`
}

// TopicKafka represents Kafka-specific metadata
type TopicKafka struct {
	Partitions int               `json:"partitions"`
	Configs    *TopicKafkaConfig `json:"configs,omitempty"`
}

// TopicDetailed represents the full topic response from /topics/{id}?detailed=true
type TopicDetailed struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Entity        *TopicEntity `json:"entity,omitempty"`
	Kafka         *TopicKafka  `json:"kafka,omitempty"`
	Prefix        *string      `json:"prefix,omitempty"`
	Serialization *string      `json:"serialization,omitempty"`
}


func (s *streamkapAPI) UpdateTopic(ctx context.Context, topicID string, reqPayload Topic) (*Topic, error) {
	// Backend expects {"payload": {"partition_count": ..., "tags": [...]}}
	// where at least one of partition_count/tags is required.
	innerPayload := map[string]any{
		"partition_count": reqPayload.PartitionCount,
	}
	if reqPayload.Tags != nil {
		innerPayload["tags"] = reqPayload.Tags
	}
	expectedPayload := map[string]any{"payload": innerPayload}
	payload, err := json.Marshal(expectedPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/topics/"+topicID, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateTopic request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))
	var rep any
	err = s.doRequestWithRetry(ctx, req, &rep)
	if err != nil {
		return nil, err
	}

	return &reqPayload, nil
}

func (s *streamkapAPI) GetTopic(ctx context.Context, topicID string) (*Topic, error) {
	// Backend returns full TopicDetailsWithKafka by default (detailed=true is the
	// default). We rely on the `kafka.partitions.count` nested field — pin
	// detailed=true explicitly so a future backend default flip can't silently
	// strip the partition count from our Read path.
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, s.cfg.BaseURL+"/topics/"+topicID+"?detailed=true", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTopic request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))

	var resp Topic
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	// Lift the live partition count out of the nested kafka block so callers
	// can keep treating Topic.PartitionCount as a simple top-level field.
	if resp.Kafka != nil {
		resp.PartitionCount = resp.Kafka.Partitions.Count
	}

	return &resp, nil
}

func (s *streamkapAPI) DeleteTopic(ctx context.Context, topicID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/topics/"+topicID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteTopic request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Topic
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (s *streamkapAPI) ListTopics(ctx context.Context, params *TopicListParams) (*TopicDetailsResponse, error) {
	url := s.cfg.BaseURL + "/topics/details"

	// Build query parameters
	queryParams := make([]string, 0)
	if params != nil {
		if params.EntityType != "" {
			queryParams = append(queryParams, "entity_type="+params.EntityType)
		}
		if len(params.EntityIDs) > 0 {
			for _, id := range params.EntityIDs {
				queryParams = append(queryParams, "entity_id="+id)
			}
		}
		if params.Limit > 0 {
			queryParams = append(queryParams, fmt.Sprintf("limit=%d", params.Limit))
		}
		if params.Offset > 0 {
			queryParams = append(queryParams, fmt.Sprintf("offset=%d", params.Offset))
		}
	}
	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"ListTopics request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))

	var resp TopicDetailsResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) GetTopicTableMetrics(ctx context.Context, reqPayload TopicTableMetricsRequest) (TopicTableMetricsResponse, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, s.cfg.BaseURL+"/topics/table_metrics", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTopicTableMetrics request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))

	var resp TopicTableMetricsResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *streamkapAPI) GetTopicDetailed(ctx context.Context, topicID string) (*TopicDetailed, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, s.cfg.BaseURL+"/topics/"+topicID+"?detailed=true", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTopicDetailed request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))

	var resp TopicDetailed
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
