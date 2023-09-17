package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type CreateDestinationRequest struct {
	Name      string `json:"name"`
	Connector string `json:"connector"`
	Config    struct {
		DatabaseHostnameUserDefined string `json:"database.hostname.user.defined"`
		DatabasePortUserDefined     string `json:"database.port.user.defined"`
		DatabaseDatabaseUserDefined string `json:"database.database.user.defined"`
		ConnectionUsername          string `json:"connection.username"`
		ConnectionPassword          string `json:"connection.password"`
		DeleteEnabled               bool   `json:"delete.enabled"`
		InsertMode                  string `json:"insert.mode"`
		SchemaEvolution             string `json:"schema.evolution"`
		TasksMax                    int    `json:"tasks.max"`
		PrimaryKeyMode              string `json:"primary.key.mode"`
		PrimaryKeyFields            string `json:"primary.key.fields"`
	} `json:"config"`
}

type CreateDestinationResponse struct {
	Cursor struct {
		TimestampFrom time.Time `json:"timestamp_from"`
		TimestampTo   time.Time `json:"timestamp_to"`
		TopicListFrom int       `json:"topic_list_from"`
		TopicListSize int       `json:"topic_list_size"`
		TopicListSort string    `json:"topic_list_sort"`
	} `json:"cursor"`
	EntityType string `json:"entity_type"`
	Data       []struct {
		Id           string `json:"id"`
		Name         string `json:"name"`
		SubId        string `json:"sub_id"`
		TenantId     string `json:"tenant_id"`
		Connector    string `json:"connector"`
		TaskStatuses struct {
			Field1 struct {
				Status string `json:"status"`
			} `json:"0"`
		} `json:"task_statuses"`
		Tasks           []int    `json:"tasks"`
		ConnectorStatus string   `json:"connector_status"`
		TopicIds        []string `json:"topic_ids"`
		Topics          []string `json:"topics"`
		InlineMetrics   struct {
			ConnectorStatus []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"connector_status"`
			Latency []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"latency"`
			SourceRecordWriteTotal []struct {
				Timestamp string `json:"timestamp"`
				Value     int    `json:"value"`
			} `json:"sourceRecordWriteTotal"`
		} `json:"inline_metrics"`
		Config struct {
			Key string `json:"key"`
		} `json:"config"`
	} `json:"data"`
}

func (s *streamkapAPI) CreateDestination(ctx context.Context, reqPayload CreateDestinationRequest) (*CreateDestinationResponse, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/destinations", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp CreateDestinationResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
