package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func TestAPISourceAndTopicDestinationLifecycle(t *testing.T) {
	const sourceID = "507f1f77bcf86cd799439011"
	var mu sync.Mutex
	var source *api.Source
	links := map[string]bool{}
	var updateResources []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		write := func(value any) { _ = json.NewEncoder(w).Encode(value) }
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/access-token":
			write(api.Token{AccessToken: "local-token", ExpiresIn: 3600})
		case r.Method == http.MethodPost && r.URL.Path == "/sources":
			var body api.Source
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode source create: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.Connector != "hubspot" || body.KcClusterId != "" {
				t.Errorf("unexpected source create body: connector=%q cluster=%q", body.Connector, body.KcClusterId)
			}
			body.ID = sourceID
			body.KcClusterId = "api-cluster"
			body.ConnectorStatus = "Pending"
			source = &body
			write(source)
		case r.Method == http.MethodGet && r.URL.Path == "/sources/"+sourceID:
			if source == nil {
				write(api.GetSourceResponse{})
				return
			}
			copy := *source
			if items, ok := copy.Config["resources"].([]any); ok {
				values := make([]string, len(items))
				for i, item := range items {
					values[i], _ = item.(string)
				}
				copy.Config = make(map[string]any, len(source.Config))
				for key, value := range source.Config {
					copy.Config[key] = value
				}
				copy.Config["resources"] = strings.Join(values, ",")
			}
			write(api.GetSourceResponse{Total: 1, Result: []api.Source{copy}})
		case r.Method == http.MethodPut && r.URL.Path == "/sources/"+sourceID:
			var body api.Source
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode source update: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if items, ok := body.Config["resources"].([]any); ok {
				for _, item := range items {
					updateResources = append(updateResources, item.(string))
				}
			}
			body.ID = sourceID
			body.KcClusterId = "api-cluster"
			body.ConnectorStatus = "Pending Update"
			source = &body
			write(source)
		case r.Method == http.MethodDelete && r.URL.Path == "/sources/"+sourceID:
			source = nil
			write(map[string]bool{"deleted": true})
		case strings.HasPrefix(r.URL.Path, "/topics/") && strings.HasSuffix(r.URL.Path, "/destinations/destination-1"):
			parts := strings.Split(r.URL.Path, "/")
			topicID := parts[2]
			if source == nil {
				w.WriteHeader(http.StatusNotFound)
				write(map[string]string{"detail": "source not found"})
				return
			}
			switch r.Method {
			case http.MethodPut:
				links[topicID] = true
			case http.MethodGet:
				if !links[topicID] {
					w.WriteHeader(http.StatusNotFound)
					write(map[string]string{"detail": "link not found"})
					return
				}
			case http.MethodDelete:
				delete(links, topicID)
				write(map[string]bool{"detached": true})
				return
			}
			write(api.TopicDestinationLink{BindingID: "binding-1", SourceID: sourceID, DestinationID: "destination-1", TopicIDs: []string{topicID}})
		default:
			t.Errorf("unexpected API request %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			write(map[string]string{"detail": "unexpected request"})
		}
	}))
	defer server.Close()

	providerConfig := fmt.Sprintf(`provider "streamkap" {
  host = %q
  client_id = "local-client"
  secret = "local-secret"
}
`, server.URL)
	config := func(resources string, includeContacts bool) string {
		contactLink := ""
		if includeContacts {
			contactLink = `resource "streamkap_topic_destination" "contacts" {
	  topic_id = "source_${streamkap_source_hubspot.crm.id}.hubspot.contacts"
  destination_id = "destination-1"
}
`
		}
		return providerConfig + fmt.Sprintf(`resource "streamkap_source_hubspot" "crm" {
  name = "crm"
  token = "local-vendor-token"
  resources = %s
}
%s
resource "streamkap_topic_destination" "companies" {
	  topic_id = "source_${streamkap_source_hubspot.crm.id}.hubspot.companies"
  destination_id = "destination-1"
}
`, resources, contactLink)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`["contacts", "companies"]`, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_hubspot.crm", "kc_cluster_id", "api-cluster"),
					resource.TestCheckResourceAttr("streamkap_topic_destination.contacts", "binding_id", "binding-1"),
					resource.TestCheckResourceAttr("streamkap_topic_destination.companies", "binding_id", "binding-1"),
				),
			},
			{
				ResourceName: "streamkap_source_hubspot.crm", ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			{
				ResourceName: "streamkap_topic_destination.contacts", ImportState: true, ImportStateVerify: true,
			},
			{
				Config: config(`["contacts", "companies", "deals"]`, true),
				Check:  resource.TestCheckResourceAttr("streamkap_source_hubspot.crm", "resources.#", "3"),
			},
			{
				Config: config(`["contacts", "companies", "deals"]`, false),
				Check:  resource.TestCheckResourceAttr("streamkap_topic_destination.companies", "binding_id", "binding-1"),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if source != nil || len(links) != 0 {
		t.Errorf("destroy left source=%v links=%v", source != nil, links)
	}
	if strings.Join(updateResources, ",") != "contacts,companies,deals" {
		t.Errorf("source update resources = %v", updateResources)
	}
}
