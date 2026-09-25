package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTopicDestinationRequestsArePerTopic(t *testing.T) {
	var calls []string
	attached := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		parts := strings.Split(r.URL.Path, "/")
		topicID := parts[2]
		switch r.Method {
		case http.MethodPut:
			attached[topicID] = true
		case http.MethodGet:
			if !attached[topicID] {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"detail":"not found"}`))
				return
			}
		case http.MethodDelete:
			delete(attached, topicID)
			w.Write([]byte(`{"detached":true}`))
			return
		}
		topics := make([]string, 0, len(attached))
		for topic := range attached {
			topics = append(topics, topic)
		}
		json.NewEncoder(w).Encode(TopicDestinationLink{BindingID: "binding-1", SourceID: "source-1", DestinationID: "dest-1", TopicIDs: topics})
	}))
	defer server.Close()
	client := NewClient(&Config{BaseURL: server.URL})
	client.SetToken(&Token{AccessToken: "test-token"})
	ctx := context.Background()
	if _, err := client.AttachTopicDestination(ctx, "source_1.Account", "dest-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AttachTopicDestination(ctx, "source_1.Contact", "dest-1"); err != nil {
		t.Fatal(err)
	}
	link, err := client.GetTopicDestination(ctx, "source_1.Contact", "dest-1")
	if err != nil || link.BindingID != "binding-1" || len(link.TopicIDs) != 2 {
		t.Fatalf("GetTopicDestination = %#v, %v", link, err)
	}
	if err := client.DetachTopicDestination(ctx, "source_1.Account", "dest-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetTopicDestination(ctx, "source_1.Contact", "dest-1"); err != nil {
		t.Fatalf("detaching Account removed Contact from the shared binding: %v", err)
	}
	want := []string{
		"PUT /topics/source_1.Account/destinations/dest-1",
		"PUT /topics/source_1.Contact/destinations/dest-1",
		"GET /topics/source_1.Contact/destinations/dest-1",
		"DELETE /topics/source_1.Account/destinations/dest-1",
		"GET /topics/source_1.Contact/destinations/dest-1",
	}
	for i, call := range calls {
		if i >= len(want) || call != want[i] {
			t.Fatalf("calls = %v, want %v", calls, want)
		}
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}
