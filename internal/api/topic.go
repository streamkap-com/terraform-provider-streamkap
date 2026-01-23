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

type Topic struct {
	TopicID        string `json:"topic_id"`
	PartitionCount int    `json:"partition_count"`
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


func (s *streamkapAPI) UpdateTopic(ctx context.Context, topicID string, reqPayload Topic) (*Topic, error) {
	expectedPayload := map[string]map[string]int{
		"payload": {},
	}
	expectedPayload["payload"]["partition_count"] = reqPayload.PartitionCount
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
		payload,
	))
	var rep any
	err = s.doRequestWithRetry(ctx, req, &rep)
	if err != nil {
		return nil, err
	}

	return &reqPayload, nil
}

func (s *streamkapAPI) GetTopic(ctx context.Context, topicID string) (*Topic, error) {

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, s.cfg.BaseURL+"/topics/"+topicID, http.NoBody)
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
		payload,
	))

	var resp TopicTableMetricsResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
