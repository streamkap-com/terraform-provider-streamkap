package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type TopicDestinationLink struct {
	BindingID     string   `json:"binding_id"`
	SourceID      string   `json:"source_id"`
	DestinationID string   `json:"destination_id"`
	TopicIDs      []string `json:"topic_ids"`
}

func (s *streamkapAPI) topicDestinationURL(topicID, destinationID string) string {
	return fmt.Sprintf("%s/topics/%s/destinations/%s", s.cfg.BaseURL, url.PathEscape(topicID), url.PathEscape(destinationID))
}

func (s *streamkapAPI) topicDestinationRequest(ctx context.Context, method, topicID, destinationID string, result any) error {
	req, err := http.NewRequestWithContext(ctx, method, s.topicDestinationURL(topicID, destinationID), http.NoBody)
	if err != nil {
		return fmt.Errorf("build %s topic destination request: %w", method, err)
	}
	if method == http.MethodGet {
		return s.doRequest(ctx, req, result)
	}
	return s.doRequestWithRetry(ctx, req, result)
}

func (s *streamkapAPI) AttachTopicDestination(ctx context.Context, topicID, destinationID string) (*TopicDestinationLink, error) {
	var link TopicDestinationLink
	if err := s.topicDestinationRequest(ctx, http.MethodPut, topicID, destinationID, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *streamkapAPI) GetTopicDestination(ctx context.Context, topicID, destinationID string) (*TopicDestinationLink, error) {
	var link TopicDestinationLink
	if err := s.topicDestinationRequest(ctx, http.MethodGet, topicID, destinationID, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *streamkapAPI) DetachTopicDestination(ctx context.Context, topicID, destinationID string) error {
	return s.deleteResource(ctx, "DetachTopicDestination", s.topicDestinationURL(topicID, destinationID))
}
