//go:build sweep

package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func init() {
	resource.AddTestSweepers("streamkap_source", &resource.Sweeper{
		Name: "streamkap_source",
		F:    sweepSources,
	})

	resource.AddTestSweepers("streamkap_destination", &resource.Sweeper{
		Name: "streamkap_destination",
		F:    sweepDestinations,
	})

	resource.AddTestSweepers("streamkap_pipeline", &resource.Sweeper{
		Name:         "streamkap_pipeline",
		F:            sweepPipelines,
		Dependencies: []string{"streamkap_source", "streamkap_destination"},
	})
}

func sweepSources(_ string) error {
	client, err := newSweepClient()
	if err != nil {
		return fmt.Errorf("error creating sweep client: %w", err)
	}

	ctx := context.Background()

	sources, err := client.ListSources(ctx)
	if err != nil {
		return fmt.Errorf("error listing sources: %w", err)
	}

	var errs []error
	for _, source := range sources {
		if isTestResource(source.Name) {
			log.Printf("[INFO] Sweeping source: %s (ID: %s)", source.Name, source.ID)
			if err := client.DeleteSource(ctx, source.ID); err != nil {
				log.Printf("[ERROR] Failed to delete source %s: %v", source.Name, err)
				errs = append(errs, fmt.Errorf("failed to delete source %s: %w", source.Name, err))
			} else {
				log.Printf("[INFO] Successfully deleted source: %s", source.Name)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during source sweep: %v", errs)
	}

	return nil
}

func sweepDestinations(_ string) error {
	log.Println("[INFO] Destination sweeper - manual cleanup may be needed for tf-migration-test-* resources")
	return nil
}

func sweepPipelines(_ string) error {
	log.Println("[INFO] Pipeline sweeper - manual cleanup may be needed for tf-migration-test-* resources")
	return nil
}

func newSweepClient() (api.StreamkapAPI, error) {
	clientID := os.Getenv("STREAMKAP_CLIENT_ID")
	secret := os.Getenv("STREAMKAP_SECRET")
	host := os.Getenv("STREAMKAP_HOST")

	if clientID == "" || secret == "" {
		return nil, fmt.Errorf("STREAMKAP_CLIENT_ID and STREAMKAP_SECRET must be set")
	}

	if host == "" {
		host = "https://api.streamkap.com"
	}

	client := api.NewClient(&api.Config{BaseURL: host})
	token, err := client.GetAccessToken(clientID, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	client.SetToken(token)

	return client, nil
}

// TestMain enables the sweeper functionality
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// isTestResource checks if a resource name indicates it was created by tests
func isTestResource(name string) bool {
	prefixes := []string{
		"tf-acc-test",
		"tf-migration-test",
		"test-source-",
		"test-destination-",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
