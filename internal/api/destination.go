package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type GetDestinationResponse struct {
	Total    int           `json:"total"`
	PageSize int           `json:"page_size"`
	Page     int           `json:"page"`
	Result   []Destination `json:"result"`
}

type Destination struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Connector string         `json:"connector"`
	Config    map[string]any `json:"config"`
}

func (s *streamkapAPI) CreateDestination(ctx context.Context, reqPayload Destination) (*Destination, error) {
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

func (s *streamkapAPI) GetDestination(ctx context.Context, destinationID string) (*Destination, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/destinations?secret_returned=true&id="+destinationID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp GetDestinationResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) DeleteDestination(ctx context.Context, destinationID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/api/destinations?secret_returned=true&id="+destinationID, http.NoBody)
	if err != nil {
		return err
	}
	var resp Destination
	err = s.doRequest(req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdateDestination(ctx context.Context, destinationID string, reqPayload Destination) (*Destination, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/api/destinations?secret_returned=true&id="+destinationID, bytes.NewBuffer(payload))
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
