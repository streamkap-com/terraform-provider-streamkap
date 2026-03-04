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

type Role struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ClientCredential struct {
	ClientID    string `json:"client_id"`
	Secret      string `json:"secret,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	Roles       []Role `json:"roles"`
	ServiceID   string `json:"service_id,omitempty"`
}

type CreateClientCredentialRequest struct {
	RoleIDs     []string `json:"role_ids"`
	Description string   `json:"description,omitempty"`
	ServiceID   string   `json:"service_id,omitempty"`
}

func (s *streamkapAPI) CreateClientCredential(ctx context.Context, reqPayload CreateClientCredentialRequest) (*ClientCredential, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/auth/client-credentials", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateClientCredential request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp ClientCredential
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *streamkapAPI) GetClientCredential(ctx context.Context, clientID string) (*ClientCredential, error) {
	credentials, err := s.ListClientCredentials(ctx)
	if err != nil {
		return nil, err
	}

	for _, cred := range credentials {
		if cred.ClientID == clientID {
			return &cred, nil
		}
	}

	return nil, nil
}

func (s *streamkapAPI) ListClientCredentials(ctx context.Context) ([]ClientCredential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/auth/client-credentials", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"ListClientCredentials request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp []ClientCredential
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *streamkapAPI) DeleteClientCredential(ctx context.Context, clientID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/auth/client-credentials/"+clientID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteClientCredential request details:\n"+
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
