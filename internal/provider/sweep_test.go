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

	resource.AddTestSweepers("streamkap_transform", &resource.Sweeper{
		Name: "streamkap_transform",
		F:    sweepTransforms,
	})

	resource.AddTestSweepers("streamkap_pipeline", &resource.Sweeper{
		Name:         "streamkap_pipeline",
		F:            sweepPipelines,
		Dependencies: []string{"streamkap_source", "streamkap_destination", "streamkap_transform"},
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
	client, err := newSweepClient()
	if err != nil {
		return fmt.Errorf("error creating sweep client: %w", err)
	}

	ctx := context.Background()

	destinations, err := client.ListDestinations(ctx)
	if err != nil {
		return fmt.Errorf("error listing destinations: %w", err)
	}

	var errs []error
	for _, destination := range destinations {
		if isTestResource(destination.Name) {
			log.Printf("[INFO] Sweeping destination: %s (ID: %s)", destination.Name, destination.ID)
			if err := client.DeleteDestination(ctx, destination.ID); err != nil {
				log.Printf("[ERROR] Failed to delete destination %s: %v", destination.Name, err)
				errs = append(errs, fmt.Errorf("failed to delete destination %s: %w", destination.Name, err))
			} else {
				log.Printf("[INFO] Successfully deleted destination: %s", destination.Name)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during destination sweep: %v", errs)
	}

	return nil
}

func sweepTransforms(_ string) error {
	client, err := newSweepClient()
	if err != nil {
		return fmt.Errorf("error creating sweep client: %w", err)
	}

	ctx := context.Background()

	transforms, err := client.ListTransforms(ctx)
	if err != nil {
		return fmt.Errorf("error listing transforms: %w", err)
	}

	var errs []error
	for _, transform := range transforms {
		if isTestResource(transform.Name) {
			log.Printf("[INFO] Sweeping transform: %s (ID: %s)", transform.Name, transform.ID)
			if err := client.DeleteTransform(ctx, transform.ID); err != nil {
				log.Printf("[ERROR] Failed to delete transform %s: %v", transform.Name, err)
				errs = append(errs, fmt.Errorf("failed to delete transform %s: %w", transform.Name, err))
			} else {
				log.Printf("[INFO] Successfully deleted transform: %s", transform.Name)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during transform sweep: %v", errs)
	}

	return nil
}

func sweepPipelines(_ string) error {
	client, err := newSweepClient()
	if err != nil {
		return fmt.Errorf("error creating sweep client: %w", err)
	}

	ctx := context.Background()

	pipelines, err := client.ListPipelines(ctx)
	if err != nil {
		return fmt.Errorf("error listing pipelines: %w", err)
	}

	var errs []error
	for _, pipeline := range pipelines {
		if isTestResource(pipeline.Name) {
			log.Printf("[INFO] Sweeping pipeline: %s (ID: %s)", pipeline.Name, pipeline.ID)
			if err := client.DeletePipeline(ctx, pipeline.ID); err != nil {
				log.Printf("[ERROR] Failed to delete pipeline %s: %v", pipeline.Name, err)
				errs = append(errs, fmt.Errorf("failed to delete pipeline %s: %w", pipeline.Name, err))
			} else {
				log.Printf("[INFO] Successfully deleted pipeline: %s", pipeline.Name)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during pipeline sweep: %v", errs)
	}

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
