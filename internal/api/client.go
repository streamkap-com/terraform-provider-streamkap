package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type StreamkapAPI interface {
	GetAccessToken(clientID, secret string) (*Token, error)
	SetToken(token *Token)

	//Source APIs
	ListSourceConfigurations(ctx context.Context) ([]SourceConfigurationResponse, error)
	CreateSource(ctx context.Context, req CreateSourceRequest) (*CreateSourceResponse, error)
	GetSource(ctx context.Context, sourceID string) (*Source, error)
	CreateDestination(ctx context.Context, reqPayload CreateDestinationRequest) (*CreateDestinationResponse, error)
	CreatePipeline(ctx context.Context, reqPayload CreatePipelineRequest) (*CreatePipelineResponse, error)
}

type Config struct {
	BaseURL string `mapstructure:"base_url"`
}

type streamkapAPI struct {
	cfg    *Config
	client *http.Client
	token  *Token
}

func NewClient(cfg *Config) StreamkapAPI {
	return &streamkapAPI{
		cfg:    cfg,
		client: http.DefaultClient,
	}
}

func (s *streamkapAPI) SetToken(token *Token) {
	s.token = token
}

func (s *streamkapAPI) doRequest(req *http.Request, result interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if s.token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token.AccessToken))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}
