package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Source struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SubID        string `json:"sub_id"`
	TenantID     string `json:"tenant_id"`
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
		SnapshotState []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"snapshotState"`
		State []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"state"`
		StreamingState []struct {
			Timestamp string `json:"timestamp"`
			Value     string `json:"value"`
		} `json:"streamingState"`
		SourceRecordWriteTotal []struct {
			Timestamp string `json:"timestamp"`
			Value     int    `json:"value"`
		} `json:"sourceRecordWriteTotal"`
	} `json:"inline_metrics"`
	Server string `json:"server"`
	Config struct {
		Key string `json:"key"`
	} `json:"config"`
}

type CreateSourceRequest struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name"`
	Connector string          `json:"connector"`
	Config    json.RawMessage `json:"config"`
}

type CreateSourceResponse struct {
	Cursor struct {
		TimestampFrom time.Time `json:"timestamp_from"`
		TimestampTo   time.Time `json:"timestamp_to"`
		TopicListFrom int       `json:"topic_list_from"`
		TopicListSize int       `json:"topic_list_size"`
		TopicListSort string    `json:"topic_list_sort"`
	} `json:"cursor"`
	EntityType string   `json:"entity_type"`
	Data       []Source `json:"data"`
}

func (s *streamkapAPI) CreateSource(ctx context.Context, reqPayload CreateSourceRequest) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/sources", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) GetSource(ctx context.Context, sourceID string) ([]Source, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/sources/"+sourceID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp []Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *streamkapAPI) DeleteSource(ctx context.Context, sourceID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/api/sources/"+sourceID, http.NoBody)
	if err != nil {
		return err
	}
	var resp Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdateSource(ctx context.Context, reqPayload CreateSourceRequest) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/api/sources", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
