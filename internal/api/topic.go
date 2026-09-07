package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	PartitionCount int         `json:"-"`
	Tags           []string    `json:"tags"`
	Kafka          *TopicKafka `json:"kafka,omitempty"`
}

// TopicPartitionsInfo carries the live partition count from the broker.
type TopicPartitionsInfo struct {
	Count int `json:"count"`
}

// TopicEntity represents the entity (source/transform/destination) that owns a topic
type TopicEntity struct {
	EntityType  string   `json:"entity_type"` // "sources", "transforms", "destinations"
	EntityID    string   `json:"entity_id"`
	Name        string   `json:"name"`
	Connector   string   `json:"connector"`
	DisplayName string   `json:"display_name"`
	TopicIDs    []string `json:"topic_ids"`
	TopicDBIDs  []string `json:"topic_db_ids"`
}

// TopicSerialization mirrors the backend's TopicSerialization object. The
// backend changed the `serialization` field on /topics responses from a plain
// string to this object; KeyFormat/ValueFormat carry the human-readable format
// (avro, json, json_schema, protobuf, string, bytearray, unknown) and the
// converter fields carry the underlying Kafka Connect converter class names.
type TopicSerialization struct {
	KeyFormat             string  `json:"key_format"`
	ValueFormat           string  `json:"value_format"`
	KeyConverter          *string `json:"key_converter,omitempty"`
	ValueConverter        *string `json:"value_converter,omitempty"`
	SchemaRegistryEnabled bool    `json:"schema_registry_enabled"`
}

// TopicDetails represents detailed topic information from /topics/details
type TopicDetails struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Entity        *TopicEntity        `json:"entity,omitempty"`
	Prefix        *string             `json:"prefix,omitempty"`
	Serialization *TopicSerialization `json:"serialization,omitempty"`
	// The topic details endpoint declares neither field and the backend has no
	// such key, so both always decode to nil. Kept so the data source keeps
	// parsing; see the deprecation notices on the Terraform attributes.
	Messages7D  *int64 `json:"messages_7d,omitempty"`
	Messages30D *int64 `json:"messages_30d,omitempty"`
}

// TopicDetailsResponse represents the paginated response from /topics/details
type TopicDetailsResponse struct {
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int            `json:"total"`
	HasNext  bool           `json:"has_next"`
	Result   []TopicDetails `json:"result"`
}

// TopicListParams represents query parameters for listing topics.
// There is deliberately no limit/offset: the backend's TopicDetailsReq declares
// only page/page_size, so limit/offset were accepted and ignored. ListTopics
// paginates internally and returns every match.
type TopicListParams struct {
	EntityType string   // Filter by entity type: "sources", "transforms", "destinations"
	EntityIDs  []string // Filter by specific entity IDs
}

// topicsSearchBody mirrors the backend's TopicDetailsReqBody for
// POST /topics/details/search. entity_id accepts an array or a comma-separated
// string; body values take precedence over query params.
type topicsSearchBody struct {
	EntityID []string `json:"entity_id,omitempty"`
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
	Entities []TopicMetricsEntity `json:"entities"`
}

type TopicTableKafkaMetrics struct {
	PartitionCount    *int64 `json:"partition_count"`
	ReplicationFactor *int64 `json:"replication_factor"`
	RetentionMs       *int64 `json:"retention_ms"`
}

// TopicTableMetricsRow mirrors one value in the topic-id-keyed response from
// POST /topics/table_metrics. Broker and ClickHouse values are nullable because
// either subsystem may have no data for a valid topic.
type TopicTableMetricsRow struct {
	ID                   string                 `json:"id"`
	Kafka                TopicTableKafkaMetrics `json:"kafka"`
	LastMessageTimestamp *int64                 `json:"lastMessageTimestamp"`
	SnapshotStatus       []map[string]any       `json:"snapshotStatus"`
	RecordErrorTotal     *int64                 `json:"recordErrorTotal"`
}

// TopicTableMetricsResponse is keyed by Kafka topic ID.
type TopicTableMetricsResponse map[string]TopicTableMetricsRow

// TopicKafkaConfig mirrors the backend's TopicKafkaConfigs. The keys are
// underscored field names carrying strings, not the dotted Kafka property names
// (`retention.ms`) the broker itself uses — the backend renames and stringifies
// them on the way out.
type TopicKafkaConfig struct {
	RetentionMs   *string `json:"retention_ms,omitempty"`
	CleanupPolicy *string `json:"cleanup_policy,omitempty"`
}

// TopicKafka mirrors the slice of the backend's TopicKafkaMetadata the provider
// reads. `partitions` is an object (TopicKafkaPartitions), so decoding it as a
// bare count fails the whole response.
type TopicKafka struct {
	Partitions TopicPartitionsInfo `json:"partitions"`
	Configs    *TopicKafkaConfig   `json:"configs,omitempty"`
}

// TopicDetailed represents the full topic response from /topics/{id}?detailed=true
type TopicDetailed struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Entity        *TopicEntity        `json:"entity,omitempty"`
	Kafka         *TopicKafka         `json:"kafka,omitempty"`
	Prefix        *string             `json:"prefix,omitempty"`
	Serialization *TopicSerialization `json:"serialization,omitempty"`
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
	// The backend defaults to detailed=false, which omits the nested kafka
	// block. Read relies on `kafka.partitions.count`, so detailed=true is
	// required here, not merely a defensive pin.
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

// ListTopics returns every topic matching params. The backend defaults to
// page_size=10 (max 100), so it iterates pages until a short one comes back —
// the same pattern as ListPipelines, with the same runaway guard. The returned
// response describes the aggregate: Result holds all pages, Total is the count
// the backend reported for the query.
func (s *streamkapAPI) ListTopics(ctx context.Context, params *TopicListParams) (*TopicDetailsResponse, error) {
	const pageSize = 100
	const maxPages = 1000

	var all []TopicDetails
	total := 0
	for page := 1; page <= maxPages; page++ {
		req, err := s.newTopicDetailsRequest(ctx, params, page, pageSize)
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
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Result...)
		total = resp.Total

		// Short-page termination only; `total` can lie under concurrent topic
		// creation, and maxPages caps the runaway. Same reasoning as ListSources.
		if len(resp.Result) < pageSize {
			break
		}
	}

	// Page/PageSize/HasNext describe one backend page and are meaningless once
	// every page has been merged, so they are left at zero rather than filled
	// with invented values (PageSize is a request parameter, not a result count).
	// The only consumer, the topics data source, reads Total and Result.
	return &TopicDetailsResponse{
		Total:  total,
		Result: all,
	}, nil
}

// newTopicDetailsRequest builds one page request for /topics/details.
//
// entity_id is a single scalar query param that the backend splits on commas —
// sending it as a repeated key made a multi-element filter collapse to one
// arbitrary ID. When the joined list would blow up the URL, the request is
// re-routed to POST /topics/details/search, which carries the same filters in a
// body (the backend documents it for exactly this case).
func (s *streamkapAPI) newTopicDetailsRequest(ctx context.Context, params *TopicListParams, page, pageSize int) (*http.Request, error) {
	parsedURL, err := url.Parse(s.cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("ListTopics: invalid base URL %q: %w", s.cfg.BaseURL, err)
	}
	parsedURL = parsedURL.JoinPath("topics", "details")

	q := parsedURL.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))

	var entityIDs []string
	if params != nil {
		if params.EntityType != "" {
			q.Set("entity_type", params.EntityType)
		}
		entityIDs = params.EntityIDs
	}
	if len(entityIDs) > 0 {
		q.Set("entity_id", strings.Join(entityIDs, ","))
	}
	parsedURL.RawQuery = q.Encode()

	if len(parsedURL.String()) <= listURLLengthThreshold {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("ListTopics: failed to build the request: %w", err)
		}
		return req, nil
	}

	// The entity IDs move into the body; the other filters stay on the query
	// string, which the search endpoint reads exactly like the GET does.
	q.Del("entity_id")
	searchURL := *parsedURL
	searchURL.Path += "/search"
	searchURL.RawQuery = q.Encode()

	body, err := json.Marshal(topicsSearchBody{EntityID: entityIDs})
	if err != nil {
		return nil, fmt.Errorf("ListTopics: failed to encode the search body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("ListTopics: failed to build the search request: %w", err)
	}
	return req, nil
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
