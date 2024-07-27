package api

import (
	"context"
	"net/http"
)

type GetTransformResponse struct {
	Total    int         `json:"total"`
	PageSize int         `json:"page_size"`
	Page     int         `json:"page"`
	Result   []Transform `json:"result"`
}

type Transform struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name"`
	StartTime *string  `json:"start_time"`
	TopicIDs  []string `json:"topic_ids"`
	Topics    []string `json:"topics"`
}

func (s *streamkapAPI) GetTransform(ctx context.Context, TransformID string) (*Transform, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/transforms?secret_returned=true&unwind_topics=false&id="+TransformID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp GetTransformResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, nil
	}

	return &resp.Result[0], nil
}
