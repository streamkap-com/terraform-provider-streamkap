package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// apiSourceFixture mimics the backend's API-source contract closely enough to
// drive the Terraform CLI: the stored config is the stringified connect body
// (lists as CSV, booleans and numbers as strings, unset fields absent), an
// update replaces the config wholesale except for a secret it leaves out, an
// explicit null clears a secret, oauth_grant_id is consumed and never stored,
// and the vendor's normalizations are applied before the value is echoed.
type apiSourceFixture struct {
	t         *testing.T
	connector string
	secrets   []string
	normalize func(config map[string]any)
	// granted holds the credentials a submitted oauth_grant_id resolves to.
	granted map[string]any
	// modeSecrets binds a secret to the auth_mode values that use it; a stored
	// one is dropped in any other mode, as the backend does on a mode switch.
	modeSecrets map[string][]string
	// refuse, when set, answers every create with this status and detail.
	refuse  *api.APIError
	warning string

	mu      sync.Mutex
	stored  *api.Source
	creates []map[string]any
	updates []map[string]any
}

const fixtureSourceID = "507f1f77bcf86cd799439012"

func (f *apiSourceFixture) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		write := func(status int, value any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(value)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/access-token":
			write(http.StatusOK, api.Token{AccessToken: "local-token", ExpiresIn: 3600})
		case r.Method == http.MethodPost && r.URL.Path == "/sources":
			if f.refuse != nil {
				write(f.refuse.StatusCode, map[string]string{"detail": f.refuse.Detail})
				return
			}
			body := f.decode(w, r)
			if body == nil {
				return
			}
			f.creates = append(f.creates, body.Config)
			for key, value := range body.Config {
				if value == nil {
					f.t.Errorf("create sent %s as null; the backend reads an unset field from its absence", key)
				}
			}
			f.stored = &api.Source{ID: fixtureSourceID, Name: body.Name, Connector: body.Connector, Tags: body.Tags, KcClusterId: "api-cluster", ConnectorStatus: "Pending", Config: f.store(body.Config, nil)}
			response := *f.stored
			response.ConnectionWarning = f.warning
			write(http.StatusAccepted, response)
		case r.URL.Path == "/sources/"+fixtureSourceID && r.Method == http.MethodGet:
			if f.stored == nil {
				write(http.StatusOK, api.GetSourceResponse{})
				return
			}
			write(http.StatusOK, api.GetSourceResponse{Total: 1, Result: []api.Source{*f.stored}})
		case r.URL.Path == "/sources/"+fixtureSourceID && r.Method == http.MethodPut:
			body := f.decode(w, r)
			if body == nil {
				return
			}
			f.updates = append(f.updates, body.Config)
			f.stored.Name, f.stored.Tags = body.Name, body.Tags
			f.stored.Config = f.store(body.Config, f.stored.Config)
			write(http.StatusAccepted, *f.stored)
		case r.URL.Path == "/sources/"+fixtureSourceID && r.Method == http.MethodDelete:
			f.stored = nil
			write(http.StatusOK, map[string]bool{"deleted": true})
		default:
			f.t.Errorf("unexpected API request %s %s", r.Method, r.URL.Path)
			write(http.StatusNotFound, map[string]string{"detail": "unexpected request"})
		}
	}))
}

func (f *apiSourceFixture) decode(w http.ResponseWriter, r *http.Request) *api.Source {
	var body api.Source
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		f.t.Errorf("decode %s %s: %v", r.Method, r.URL.Path, err)
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if body.Connector != f.connector {
		f.t.Errorf("connector = %q, want %q", body.Connector, f.connector)
	}
	return &body
}

func (f *apiSourceFixture) store(submitted, previous map[string]any) map[string]any {
	config := map[string]any{}
	for key, value := range submitted {
		if key != "oauth_grant_id" {
			config[key] = value
		}
	}
	for _, secret := range f.secrets {
		if _, sent := submitted[secret]; !sent && previous[secret] != nil {
			config[secret] = previous[secret]
		}
	}
	if submitted["oauth_grant_id"] != nil {
		for key, value := range f.granted {
			config[key] = value
		}
	}
	for secret, modes := range f.modeSecrets {
		if !slices.Contains(modes, fmt.Sprint(config["auth_mode"])) {
			delete(config, secret)
		}
	}
	if f.normalize != nil {
		f.normalize(config)
	}
	stored := map[string]any{}
	for key, value := range config {
		switch v := value.(type) {
		case nil:
		case []any:
			items := make([]string, len(v))
			for i, item := range v {
				items[i] = fmt.Sprint(item)
			}
			stored[key] = strings.Join(items, ",")
		case bool:
			stored[key] = strconv.FormatBool(v)
		case float64:
			stored[key] = strconv.FormatInt(int64(v), 10)
		default:
			stored[key] = v
		}
	}
	return stored
}

func (f *apiSourceFixture) request(list *[]map[string]any, i int) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i >= len(*list) {
		f.t.Fatalf("expected at least %d requests, got %d", i+1, len(*list))
	}
	return (*list)[i]
}

func fixtureProvider(url string) string {
	return fmt.Sprintf(`provider "streamkap" {
  host      = %q
  client_id = "local-client"
  secret    = "local-secret"
}
`, url)
}

func (f *apiSourceFixture) checkRequest(list *[]map[string]any, i int, check func(map[string]any) error) resource.TestCheckFunc {
	return func(*terraform.State) error { return check(f.request(list, i)) }
}

// Service-account key pattern: GA4. Covers the configured date kept over its
// normalized echo, the save-time connection warning, and an update that
// leaves the unchanged key out of the body.
func TestGoogleAnalyticsSourceLifecycle(t *testing.T) {
	f := &apiSourceFixture{t: t, connector: "google_analytics", secrets: []string{"service_account_key"}, warning: "devices: the property has no data yet",
		normalize: func(config map[string]any) {
			if date, ok := config["backfill_start"].(string); ok && len(date) == len("2026-01-01") {
				config["backfill_start"] = date + "T00:00:00+00:00"
			}
		}}
	server := f.server()
	defer server.Close()
	config := func(resources string) string {
		return fixtureProvider(server.URL) + fmt.Sprintf(`resource "streamkap_source_google_analytics" "ga" {
  name                = "ga"
  property_id         = "123456789"
  service_account_key = "{\"type\":\"service_account\"}"
  resources           = %s
  backfill_start      = "2026-01-01"
}
`, resources)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`["website_overview", "devices"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_google_analytics.ga", "backfill_start", "2026-01-01"),
					resource.TestCheckResourceAttr("streamkap_source_google_analytics.ga", "api_version", "v1beta"),
					resource.TestCheckResourceAttr("streamkap_source_google_analytics.ga", "keep_empty_rows", "false"),
					resource.TestCheckResourceAttr("streamkap_source_google_analytics.ga", "kc_cluster_id", "api-cluster"),
				),
			},
			{
				ResourceName: "streamkap_source_google_analytics.ga", ImportState: true, ImportStateVerify: true,
				// Import has no configuration to keep: the stored, normalized date stands.
				ImportStateVerifyIgnore: []string{"connector_status", "backfill_start"},
			},
			{
				Config: config(`["website_overview", "devices", "events"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_google_analytics.ga", "resources.#", "3"),
					f.checkRequest(&f.updates, 0, func(body map[string]any) error {
						if _, sent := body["service_account_key"]; sent {
							return fmt.Errorf("update resent the unchanged service account key")
						}
						if body["backfill_start"] != "2026-01-01" || body["property_id"] != "123456789" {
							return fmt.Errorf("update must resend every non-secret field: %v", body)
						}
						return nil
					}),
				),
			},
		},
	})
}

// Pasted-token pattern: Facebook Ads, with Stripe's single required token
// covered by the same code path. Covers the act_ prefix the backend strips,
// and clearing an optional pair by removing it from the configuration.
func TestFacebookAdsSourceLifecycle(t *testing.T) {
	f := &apiSourceFixture{t: t, connector: "facebook_ads", secrets: []string{"access_token", "app_secret"},
		normalize: func(config map[string]any) {
			if id, ok := config["account_id"].(string); ok {
				config["account_id"] = strings.TrimPrefix(id, "act_")
			}
		}}
	server := f.server()
	defer server.Close()
	config := func(app string) string {
		return fixtureProvider(server.URL) + fmt.Sprintf(`resource "streamkap_source_facebook_ads" "meta" {
  name         = "meta"
  account_id   = "act_1234567890"
  access_token = "EAAB-local-token"
  resources    = ["campaigns", "ads"]
%s}
`, app)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("  app_id = \"42\"\n  app_secret = \"app-secret\"\n"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_facebook_ads.meta", "account_id", "act_1234567890"),
					resource.TestCheckResourceAttr("streamkap_source_facebook_ads.meta", "api_version", "v26.0"),
					resource.TestCheckResourceAttr("streamkap_source_facebook_ads.meta", "app_id", "42"),
				),
			},
			{
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("streamkap_source_facebook_ads.meta", "app_id"),
					resource.TestCheckNoResourceAttr("streamkap_source_facebook_ads.meta", "app_secret"),
					f.checkRequest(&f.updates, 0, func(body map[string]any) error {
						if value, sent := body["app_secret"]; !sent || value != nil {
							return fmt.Errorf("a removed optional secret must be cleared with an explicit null, got sent=%v value=%v", sent, value)
						}
						if _, sent := body["access_token"]; sent {
							return fmt.Errorf("update resent the unchanged access token")
						}
						return nil
					}),
				),
			},
		},
	})
}

// Multi-mode pattern: Google Ads. Covers the plan-time requirement of each
// auth mode, a mode switch that clears the previous mode's key, the dashes the
// backend strips from customer IDs, and the OAuth grant handoff.
func TestGoogleAdsSourceLifecycle(t *testing.T) {
	f := &apiSourceFixture{t: t, connector: "google_ads", secrets: []string{"client_secret", "refresh_token", "service_account_key"},
		normalize: func(config map[string]any) {
			if id, ok := config["customer_id"].(string); ok {
				config["customer_id"] = strings.ReplaceAll(id, "-", "")
			}
		},
		granted: map[string]any{"auth_mode": "oauth", "refresh_token": "granted-refresh-token"},
		modeSecrets: map[string][]string{
			"client_secret": {"oauth", "refresh_token"}, "refresh_token": {"oauth", "refresh_token"}, "service_account_key": {"service_account"},
		}}
	server := f.server()
	defer server.Close()
	config := func(auth string) string {
		return fixtureProvider(server.URL) + fmt.Sprintf(`resource "streamkap_source_google_ads" "ads" {
  name        = "ads"
  customer_id = "123-456-7890"
  resources   = ["campaign", "customer"]
%s}
`, auth)
	}
	serviceAccount := "  auth_mode = \"service_account\"\n  service_account_key = \"{\\\"type\\\":\\\"service_account\\\"}\"\n"
	refreshToken := "  auth_mode = \"refresh_token\"\n  client_id = \"client\"\n  client_secret = \"client-secret\"\n  refresh_token = \"pasted-refresh-token\"\n"
	oauth := "  auth_mode = \"oauth\"\n  client_id = \"client\"\n  client_secret = \"client-secret\"\n  oauth_grant_id = \"grant-1\"\n"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config("  auth_mode = \"refresh_token\"\n  client_id = \"client\"\n"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`client_secret is required when auth_mode is refresh_token`),
			},
			{
				Config:      config(""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`The argument "auth_mode" is required`),
			},
			{
				Config:      config("  auth_mode = \"oauth\"\n  client_id = \"client\"\n  client_secret = \"client-secret\"\n"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)Google Ads sign-in needs an OAuth grant.*start-source-oauth-connect`),
			},
			{
				Config:      config(serviceAccount + "  refresh_token = \"stale\"\n"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`refresh_token is not used when auth_mode is service_account`),
			},
			{
				Config: config(serviceAccount),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_google_ads.ads", "customer_id", "123-456-7890"),
					resource.TestCheckResourceAttr("streamkap_source_google_ads.ads", "conversion_window_days", "30"),
					f.checkRequest(&f.creates, 0, func(body map[string]any) error {
						if body["auth_mode"] != "service_account" || body["conversion_window_days"] != float64(30) {
							return fmt.Errorf("create body = %v", body)
						}
						return nil
					}),
				),
			},
			{
				Config: config(oauth),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_google_ads.ads", "auth_mode", "oauth"),
					resource.TestCheckNoResourceAttr("streamkap_source_google_ads.ads", "service_account_key"),
					resource.TestCheckResourceAttr("streamkap_source_google_ads.ads", "refresh_token", "granted-refresh-token"),
					f.checkRequest(&f.updates, 0, func(body map[string]any) error {
						if body["oauth_grant_id"] != "grant-1" {
							return fmt.Errorf("a new grant must be sent: %v", body)
						}
						return nil
					}),
				),
			},
			{
				Config: config(oauth + "  login_customer_id = \"111-222-3333\"\n"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_google_ads.ads", "refresh_token", "granted-refresh-token"),
					f.checkRequest(&f.updates, 1, func(body map[string]any) error {
						if _, sent := body["oauth_grant_id"]; sent {
							return fmt.Errorf("a spent grant must not be resent")
						}
						if _, sent := body["refresh_token"]; sent {
							return fmt.Errorf("the grant's refresh token must stay with the backend")
						}
						return nil
					}),
				),
			},
			{
				Config: config(refreshToken),
				Check: f.checkRequest(&f.updates, 2, func(body map[string]any) error {
					if body["refresh_token"] != "pasted-refresh-token" || body["auth_mode"] != "refresh_token" {
						return fmt.Errorf("the pasted refresh token must replace the granted one: %v", body)
					}
					return nil
				}),
			},
		},
	})
	if n := len(f.updates); n != 3 {
		t.Errorf("updates = %d, want 3 (every step must converge without a follow-up change)", n)
	}
}

// Zendesk signs in only through Connect, and a deployment without its OAuth
// app refuses the create with a customer-facing reason.
func TestZendeskSourceRefusedOnDeployment(t *testing.T) {
	const reason = "Zendesk isn't enabled on this Streamkap deployment yet. Contact support to turn it on."
	f := &apiSourceFixture{t: t, connector: "zendesk", refuse: &api.APIError{StatusCode: http.StatusForbidden, Detail: reason}}
	server := f.server()
	defer server.Close()
	config := func(grant string) string {
		return fixtureProvider(server.URL) + fmt.Sprintf(`resource "streamkap_source_zendesk" "support" {
  name      = "support"
  subdomain = "acme"
  resources = ["tickets"]
%s}
`, grant)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config(""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)Zendesk has no pasted-credential mode`),
			},
			{
				Config: config("  oauth_grant_id = \"grant-1\"\n"),
				// Terraform wraps the detail, so match across the line break.
				ExpectError: regexp.MustCompile(`Zendesk isn't enabled on this Streamkap deployment\s+yet\. Contact support to turn it on\. \(HTTP 403\)`),
			},
		},
	})
}
