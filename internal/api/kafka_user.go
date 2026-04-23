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

type KafkaACL struct {
	TopicName           string `json:"topic_name"`
	Operation           string `json:"operation"`
	ResourcePatternType string `json:"resource_pattern_type"`
	Resource            string `json:"resource"`
}

type KafkaUser struct {
	Username               string     `json:"username"`
	Password               string     `json:"password,omitempty"`
	WhitelistIPs           string     `json:"whitelist_ips,omitempty"`
	KafkaProxyEndpoint     string     `json:"kafka_proxy_endpoint"`
	SchemaProxyEndpoint    string     `json:"schema_proxy_endpoint,omitempty"`
	KafkaACLs              []KafkaACL `json:"kafka_acls"`
	IsCreateSchemaRegistry bool       `json:"is_create_schema_registry"`
}

type CreateKafkaUserRequest struct {
	Username               string     `json:"username"`
	Password               string     `json:"password"`
	WhitelistIPs           string     `json:"whitelist_ips,omitempty"`
	KafkaACLs              []KafkaACL `json:"kafka_acls"`
	IsCreateSchemaRegistry bool       `json:"is_create_schema_registry"`
}

type UpdateKafkaUserRequest struct {
	Password               string     `json:"password,omitempty"`
	WhitelistIPs           string     `json:"whitelist_ips,omitempty"`
	KafkaACLs              []KafkaACL `json:"kafka_acls"`
	IsCreateSchemaRegistry bool       `json:"is_create_schema_registry"`
}

func (s *streamkapAPI) CreateKafkaUser(ctx context.Context, reqPayload CreateKafkaUserRequest) (*KafkaUser, error) {
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
	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/kafka-access/kafka-users", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateKafkaUser request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))
	var resp KafkaUser
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *streamkapAPI) GetKafkaUser(ctx context.Context, username string) (*KafkaUser, error) {
	users, err := s.ListKafkaUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Username == username {
			return &user, nil
		}
	}

	return nil, nil
}

func (s *streamkapAPI) ListKafkaUsers(ctx context.Context) ([]KafkaUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/kafka-access/kafka-users", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"ListKafkaUsers request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp []KafkaUser
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *streamkapAPI) UpdateKafkaUser(ctx context.Context, username string, reqPayload UpdateKafkaUserRequest) (*KafkaUser, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/kafka-access/kafka-users/"+username, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateKafkaUser request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))
	var resp KafkaUser
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *streamkapAPI) DeleteKafkaUser(ctx context.Context, username string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/kafka-access/kafka-users/"+username, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteKafkaUser request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	// Delete returns empty 200 response
	var resp json.RawMessage
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}
	return nil
}
