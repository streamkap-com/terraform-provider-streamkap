package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Deletes on /sources, /destinations, /pipelines use &wait=false, so the
// backend removes resources asynchronously. A single GET immediately after
// Terraform's delete can still see the record, which made CheckDestroy flaky
// (e.g. "streamkap_source_dynamodb <id> still exists"). Poll for a bounded
// window instead of checking once.
const (
	destroyPollTimeout  = 90 * time.Second
	destroyPollInterval = 3 * time.Second
)

// waitForDestroyed polls fetch until it reports the resource is gone or the
// timeout elapses. fetch returns gone=true when the resource no longer exists;
// an error from fetch (e.g. a 404 once deletion completes) is treated as gone,
// matching the previous "err means not-still-exists" semantics. Returns a
// dangling-resource error only if the resource is still present at the deadline.
func waitForDestroyed(label string, fetch func() (gone bool, err error)) error {
	deadline := time.Now().Add(destroyPollTimeout)
	for {
		gone, err := fetch()
		if err != nil || gone {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s still exists after %s", label, destroyPollTimeout)
		}
		time.Sleep(destroyPollInterval)
	}
}

// testAccCheckDestroyClient creates an API client for CheckDestroy verification.
// Returns an error if required environment variables are not set.
func testAccCheckDestroyClient() (api.StreamkapAPI, error) {
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

// testAccCheckSourceDestroy verifies that all source resources have been destroyed.
// It checks the Streamkap API to confirm that source resources no longer exist.
func testAccCheckSourceDestroy(s *terraform.State) error {
	client, err := testAccCheckDestroyClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if !strings.HasPrefix(rs.Type, "streamkap_source_") {
			continue
		}

		id := rs.Primary.ID
		if err := waitForDestroyed(fmt.Sprintf("%s %s", rs.Type, id), func() (bool, error) {
			source, err := client.GetSource(ctx, id)
			return source == nil, err
		}); err != nil {
			return err
		}
	}

	return nil
}

// testAccCheckDestinationDestroy verifies that all destination resources have been destroyed.
// It checks the Streamkap API to confirm that destination resources no longer exist.
func testAccCheckDestinationDestroy(s *terraform.State) error {
	client, err := testAccCheckDestroyClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if !strings.HasPrefix(rs.Type, "streamkap_destination_") {
			continue
		}

		id := rs.Primary.ID
		if err := waitForDestroyed(fmt.Sprintf("%s %s", rs.Type, id), func() (bool, error) {
			destination, err := client.GetDestination(ctx, id)
			return destination == nil, err
		}); err != nil {
			return err
		}
	}

	return nil
}

// testAccCheckTransformDestroy verifies that all transform resources have been destroyed.
// It checks the Streamkap API to confirm that transform resources no longer exist.
func testAccCheckTransformDestroy(s *terraform.State) error {
	client, err := testAccCheckDestroyClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if !strings.HasPrefix(rs.Type, "streamkap_transform_") {
			continue
		}

		id := rs.Primary.ID
		if err := waitForDestroyed(fmt.Sprintf("%s %s", rs.Type, id), func() (bool, error) {
			transform, err := client.GetTransform(ctx, id)
			return transform == nil, err
		}); err != nil {
			return err
		}
	}

	return nil
}

// testAccCheckPipelineDestroy verifies that all pipeline resources have been destroyed.
// It checks the Streamkap API to confirm that pipeline resources no longer exist.
func testAccCheckPipelineDestroy(s *terraform.State) error {
	client, err := testAccCheckDestroyClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "streamkap_pipeline" {
			continue
		}

		id := rs.Primary.ID
		if err := waitForDestroyed(fmt.Sprintf("pipeline %s", id), func() (bool, error) {
			pipeline, err := client.GetPipeline(ctx, id)
			return pipeline == nil, err
		}); err != nil {
			return err
		}
	}

	return nil
}

// testAccCheckTopicDestroy verifies that all topic resources have been destroyed.
// It checks the Streamkap API to confirm that topic resources no longer exist.
func testAccCheckTopicDestroy(s *terraform.State) error {
	client, err := testAccCheckDestroyClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "streamkap_topic" {
			continue
		}

		id := rs.Primary.ID
		if err := waitForDestroyed(fmt.Sprintf("topic %s", id), func() (bool, error) {
			topic, err := client.GetTopic(ctx, id)
			return topic == nil, err
		}); err != nil {
			return err
		}
	}

	return nil
}

// testAccCheckTagDestroy verifies that all tag resources have been destroyed.
// It checks the Streamkap API to confirm that tag resources no longer exist.
//
//nolint:unused // Available for tag acceptance tests
func testAccCheckTagDestroy(s *terraform.State) error {
	client, err := testAccCheckDestroyClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "streamkap_tag" {
			continue
		}

		id := rs.Primary.ID
		if err := waitForDestroyed(fmt.Sprintf("tag %s", id), func() (bool, error) {
			tag, err := client.GetTag(ctx, id)
			return tag == nil, err
		}); err != nil {
			return err
		}
	}

	return nil
}
