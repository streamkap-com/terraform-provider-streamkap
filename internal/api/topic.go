package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Topic struct {
	TopicID               string      `json:"topic_id"`
	PartitionCount        int         `json:"partition_count"`
}


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
	err = s.doRequest(ctx, req, &rep)
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
