package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

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

		source, err := client.GetSource(ctx, rs.Primary.ID)
		if err == nil && source != nil {
			return fmt.Errorf("%s %s still exists", rs.Type, rs.Primary.ID)
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

		destination, err := client.GetDestination(ctx, rs.Primary.ID)
		if err == nil && destination != nil {
			return fmt.Errorf("%s %s still exists", rs.Type, rs.Primary.ID)
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

		transform, err := client.GetTransform(ctx, rs.Primary.ID)
		if err == nil && transform != nil {
			return fmt.Errorf("%s %s still exists", rs.Type, rs.Primary.ID)
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

		pipeline, err := client.GetPipeline(ctx, rs.Primary.ID)
		if err == nil && pipeline != nil {
			return fmt.Errorf("pipeline %s still exists", rs.Primary.ID)
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

		topic, err := client.GetTopic(ctx, rs.Primary.ID)
		if err == nil && topic != nil {
			return fmt.Errorf("topic %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

// testAccCheckTagDestroy verifies that all tag resources have been destroyed.
// It checks the Streamkap API to confirm that tag resources no longer exist.
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

		tag, err := client.GetTag(ctx, rs.Primary.ID)
		if err == nil && tag != nil {
			return fmt.Errorf("tag %s still exists", rs.Primary.ID)
		}
	}

	return nil
}
