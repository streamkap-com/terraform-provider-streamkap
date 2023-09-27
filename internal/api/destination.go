package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type CreateDestinationRequest struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name"`
	Connector string          `json:"connector"`
	Config    json.RawMessage `json:"config"`
}

type CreateDestinationResponse struct {
	Cursor struct {
		TimestampFrom time.Time `json:"timestamp_from"`
		TimestampTo   time.Time `json:"timestamp_to"`
		TopicListFrom int       `json:"topic_list_from"`
		TopicListSize int       `json:"topic_list_size"`
		TopicListSort string    `json:"topic_list_sort"`
	} `json:"cursor"`
	EntityType string        `json:"entity_type"`
	Data       []Destination `json:"data"`
}

type Destination struct {
	ID           string `json:"id"`
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
}

func (s *streamkapAPI) CreateDestination(ctx context.Context, reqPayload CreateDestinationRequest) (*Destination, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/destinations", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Destination
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) GetDestination(ctx context.Context, destinationID string) ([]Destination, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/destinations/"+destinationID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp []Destination
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *streamkapAPI) DeleteDestination(ctx context.Context, destinationID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/api/destinations/"+destinationID, http.NoBody)
	if err != nil {
		return err
	}
	err = s.doRequest(req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdateDestination(ctx context.Context, reqPayload CreateDestinationRequest) (*Destination, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/api/destinations", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Destination
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
