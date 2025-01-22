package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type StreamkapAPI interface {
	GetAccessToken(clientID, secret string) (*Token, error)
	SetToken(token *Token)

	//Source APIs
	CreateSource(ctx context.Context, reqPayload Source) (*Source, error)
	UpdateSource(ctx context.Context, sourceID string, reqPayload Source) (*Source, error)
	GetSource(ctx context.Context, sourceID string) (*Source, error)
	DeleteSource(ctx context.Context, sourceID string) error

	// Destination APIs
	CreateDestination(ctx context.Context, reqPayload Destination) (*Destination, error)
	UpdateDestination(ctx context.Context, destinationID string, reqPayload Destination) (*Destination, error)
	GetDestination(ctx context.Context, destinationID string) (*Destination, error)
	DeleteDestination(ctx context.Context, destinationID string) error

	// Pipeline APIs
	CreatePipeline(ctx context.Context, reqPayload Pipeline) (*Pipeline, error)
	UpdatePipeline(ctx context.Context, pipelineID string, reqPayload Pipeline) (*Pipeline, error)
	GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error)
	DeletePipeline(ctx context.Context, pipelineID string) error

	// Transform APIs
	GetTransform(ctx context.Context, transformID string) (*Transform, error)

	// Tags APIs
	GetTag(ctx context.Context, TagID string) (*Tag, error)
}

type APIErrorResponse struct {
	Detail string `json:"detail"`
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

func (s *streamkapAPI) doRequest(ctx context.Context, req *http.Request, result interface{}) error {
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

	respBodyDecoder := json.NewDecoder(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr APIErrorResponse
		if err := respBodyDecoder.Decode(&apiErr); err != nil {
			tflog.Debug(ctx,
				fmt.Sprintf("%s request to %s got status code: %d. Failed to parse API error response: %v",
					req.Method,
					req.URL,
					resp.StatusCode,
					err,
				),
			)
			return err
		} else {
			return errors.New(apiErr.Detail)
		}
	}

	if err := respBodyDecoder.Decode(result); err != nil {
		return err
	}
	return nil
}
