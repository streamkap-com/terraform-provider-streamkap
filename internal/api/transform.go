package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

type GetTransformResponse struct {
	Total    int         `json:"total"`
	PageSize int         `json:"page_size"`
	Page     int         `json:"page"`
	Result   []Transform `json:"result"`
}

type Transform struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name"`
	TransformType   string         `json:"transform_type"`
	StartTime       *string        `json:"start_time"`
	TopicIDs        []string       `json:"topic_ids"`
	Topics          []string       `json:"topics"`
	Config          map[string]any `json:"config,omitempty"`
	Implementation  map[string]any `json:"implementation,omitempty"`
	LiveVersion     *string        `json:"live_version,omitempty"`
	PreviewVersion  *string        `json:"preview_version,omitempty"`
	JobName         *string        `json:"job_name,omitempty"`
}

func (s *streamkapAPI) GetTransform(ctx context.Context, TransformID string) (*Transform, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/transforms/"+TransformID+"?secret_returned=true&unwind_topics=false", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetTransformResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, nil
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) CreateTransform(ctx context.Context, reqPayload Transform) (*Transform, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	var payloadMap map[string]any
	err = json.Unmarshal(payload, &payloadMap)
	if err != nil {
		return nil, err
	}

	payloadMap["created_from"] = constants.TERRAFORM
	// Backend expects "transform" field for library type
	payloadMap["transform"] = reqPayload.TransformType

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/transforms?secret_returned=true", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Transform
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) UpdateTransform(ctx context.Context, transformID string, reqPayload Transform) (*Transform, error) {
	reqPayload.ID = transformID
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	var payloadMap map[string]any
	err = json.Unmarshal(payload, &payloadMap)
	if err != nil {
		return nil, err
	}

	payloadMap["transform"] = reqPayload.TransformType

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/transforms?secret_returned=true", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Transform
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) DeleteTransform(ctx context.Context, transformID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/transforms?id="+transformID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Transform
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) DeployTransformLive(ctx context.Context, transformID string, versionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/transforms/"+transformID+"/deploy-job-live/"+versionID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeployTransformLive request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp map[string]any
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return err
	}

	return nil
}
