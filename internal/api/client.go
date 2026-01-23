package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ListSources(ctx context.Context) ([]Source, error)
	DeleteSource(ctx context.Context, sourceID string) error

	// Destination APIs
	CreateDestination(ctx context.Context, reqPayload Destination) (*Destination, error)
	UpdateDestination(ctx context.Context, destinationID string, reqPayload Destination) (*Destination, error)
	GetDestination(ctx context.Context, destinationID string) (*Destination, error)
	ListDestinations(ctx context.Context) ([]Destination, error)
	DeleteDestination(ctx context.Context, destinationID string) error

	// Pipeline APIs
	CreatePipeline(ctx context.Context, reqPayload Pipeline) (*Pipeline, error)
	UpdatePipeline(ctx context.Context, pipelineID string, reqPayload Pipeline) (*Pipeline, error)
	GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error)
	ListPipelines(ctx context.Context) ([]Pipeline, error)
	DeletePipeline(ctx context.Context, pipelineID string) error

	// Transform APIs
	CreateTransform(ctx context.Context, reqPayload CreateTransformRequest) (*Transform, error)
	UpdateTransform(ctx context.Context, transformID string, reqPayload UpdateTransformRequest) (*Transform, error)
	GetTransform(ctx context.Context, transformID string) (*Transform, error)
	ListTransforms(ctx context.Context) ([]Transform, error)
	DeleteTransform(ctx context.Context, transformID string) error

	// Tags APIs
	GetTag(ctx context.Context, TagID string) (*Tag, error)
	CreateTag(ctx context.Context, reqPayload Tag) (*Tag, error)
	UpdateTag(ctx context.Context, tagID string, reqPayload Tag) (*Tag, error)
	DeleteTag(ctx context.Context, tagID string) error

	// Topic APIs
	GetTopic(ctx context.Context, TopicID string) (*Topic, error)
	GetTopicDetailed(ctx context.Context, TopicID string) (*TopicDetailed, error)
	UpdateTopic(ctx context.Context, TopicID string, reqPayload Topic) (*Topic, error)
	DeleteTopic(ctx context.Context, TopicID string) error
	ListTopics(ctx context.Context, params *TopicListParams) (*TopicDetailsResponse, error)
	GetTopicTableMetrics(ctx context.Context, req TopicTableMetricsRequest) (TopicTableMetricsResponse, error)
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

// doRequestWithRetry wraps doRequest with retry logic for transient errors.
// Use this for Create/Update/Delete operations only, not for Read.
func (s *streamkapAPI) doRequestWithRetry(ctx context.Context, req *http.Request, result any) error {
	// Capture the request body for potential retries
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
	}

	cfg := DefaultRetryConfig()

	return RetryWithBackoff(ctx, cfg, func() error {
		// Restore body for each attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		return s.doRequest(ctx, req, result)
	})
}
