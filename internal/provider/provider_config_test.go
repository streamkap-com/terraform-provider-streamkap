// internal/provider/provider_config_test.go
package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// =============================================================================
// Provider Configuration Validation Tests
// =============================================================================
// These tests verify that the provider properly handles various configuration
// scenarios: missing credentials, invalid values, and environment variable fallbacks.

// TestProviderConfig_MissingClientID tests that missing client_id produces a clear error
func TestProviderConfig_MissingClientID(t *testing.T) {
	// Clear environment variables to ensure we're testing config-level validation
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	os.Unsetenv("STREAMKAP_CLIENT_ID")
	os.Setenv("STREAMKAP_SECRET", "test-secret")
	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  # client_id intentionally omitted
  secret = "test-secret"
}

# Need a data source to trigger provider configuration
data "streamkap_topics" "test" {}
`,
				ExpectError: MustCompile(`(?i)(missing.*client_id|client_id.*required|client_id.*empty)`),
			},
		},
	})
}

// TestProviderConfig_MissingSecret tests that missing secret produces a clear error
func TestProviderConfig_MissingSecret(t *testing.T) {
	// Clear environment variables to ensure we're testing config-level validation
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	os.Setenv("STREAMKAP_CLIENT_ID", "test-client-id")
	os.Unsetenv("STREAMKAP_SECRET")
	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  client_id = "test-client-id"
  # secret intentionally omitted
}

data "streamkap_topics" "test" {}
`,
				ExpectError: MustCompile(`(?i)(missing.*secret|secret.*required|secret.*empty)`),
			},
		},
	})
}

// TestProviderConfig_BothMissing tests that missing both credentials produces clear errors
func TestProviderConfig_BothMissing(t *testing.T) {
	// Clear environment variables
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	os.Unsetenv("STREAMKAP_CLIENT_ID")
	os.Unsetenv("STREAMKAP_SECRET")
	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {}

data "streamkap_topics" "test" {}
`,
				ExpectError: MustCompile(`(?i)(missing|required|empty)`),
			},
		},
	})
}

// TestProviderConfig_EmptyClientID tests that empty string client_id produces a clear error
func TestProviderConfig_EmptyClientID(t *testing.T) {
	// Clear environment variables
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	os.Unsetenv("STREAMKAP_CLIENT_ID")
	os.Setenv("STREAMKAP_SECRET", "test-secret")
	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  client_id = ""
  secret    = "test-secret"
}

data "streamkap_topics" "test" {}
`,
				ExpectError: MustCompile(`(?i)(missing|empty|required).*client_id|client_id.*(missing|empty|required)`),
			},
		},
	})
}

// TestProviderConfig_EmptySecret tests that empty string secret produces a clear error
func TestProviderConfig_EmptySecret(t *testing.T) {
	// Clear environment variables
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	os.Setenv("STREAMKAP_CLIENT_ID", "test-client-id")
	os.Unsetenv("STREAMKAP_SECRET")
	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  client_id = "test-client-id"
  secret    = ""
}

data "streamkap_topics" "test" {}
`,
				ExpectError: MustCompile(`(?i)(missing|empty|required).*secret|secret.*(missing|empty|required)`),
			},
		},
	})
}

// TestProviderConfig_EnvironmentVariableFallback tests that env vars work when config is empty
func TestProviderConfig_EnvironmentVariableFallback(t *testing.T) {
	// This test verifies environment variable fallback logic
	// Note: This is a unit test that doesn't make actual API calls
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// We can't truly test success without valid credentials,
	// but we can verify the provider accepts environment variables
	// by checking that it fails at API call time, not at config time

	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	originalHost := os.Getenv("STREAMKAP_HOST")

	// Set fake credentials via environment
	os.Setenv("STREAMKAP_CLIENT_ID", "env-client-id")
	os.Setenv("STREAMKAP_SECRET", "env-secret")
	os.Setenv("STREAMKAP_HOST", "https://invalid.example.com")

	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		} else {
			os.Unsetenv("STREAMKAP_CLIENT_ID")
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		} else {
			os.Unsetenv("STREAMKAP_SECRET")
		}
		if originalHost != "" {
			os.Setenv("STREAMKAP_HOST", originalHost)
		} else {
			os.Unsetenv("STREAMKAP_HOST")
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  # All config from environment variables
}

data "streamkap_topics" "test" {}
`,
				// If we get a connection error, env vars were properly picked up
				// (vs. "missing client_id" error if they weren't)
				ExpectError: MustCompile(`(?i)(connect|network|host|unable|dial|refused|timeout|no such host)`),
			},
		},
	})
}

// TestProviderConfig_InvalidHost tests that invalid host URL produces connection error
func TestProviderConfig_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Clear environment variables to use explicit config
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	originalHost := os.Getenv("STREAMKAP_HOST")
	os.Unsetenv("STREAMKAP_HOST")
	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		}
		if originalHost != "" {
			os.Setenv("STREAMKAP_HOST", originalHost)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  host      = "https://this-host-does-not-exist.invalid"
  client_id = "test-client-id"
  secret    = "test-secret"
}

data "streamkap_topics" "test" {}
`,
				ExpectError: MustCompile(`(?i)(unable|connect|network|dial|host|refused|no such host)`),
			},
		},
	})
}

// TestProviderConfig_ConfigOverridesEnv tests that explicit config overrides environment variables
func TestProviderConfig_ConfigOverridesEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Set environment variables that would work
	originalClientID := os.Getenv("STREAMKAP_CLIENT_ID")
	originalSecret := os.Getenv("STREAMKAP_SECRET")
	originalHost := os.Getenv("STREAMKAP_HOST")

	os.Setenv("STREAMKAP_CLIENT_ID", "env-client-id")
	os.Setenv("STREAMKAP_SECRET", "env-secret")
	os.Setenv("STREAMKAP_HOST", "https://api.streamkap.com")

	defer func() {
		if originalClientID != "" {
			os.Setenv("STREAMKAP_CLIENT_ID", originalClientID)
		} else {
			os.Unsetenv("STREAMKAP_CLIENT_ID")
		}
		if originalSecret != "" {
			os.Setenv("STREAMKAP_SECRET", originalSecret)
		} else {
			os.Unsetenv("STREAMKAP_SECRET")
		}
		if originalHost != "" {
			os.Setenv("STREAMKAP_HOST", originalHost)
		} else {
			os.Unsetenv("STREAMKAP_HOST")
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  # Override host with invalid one to prove config takes precedence
  host = "https://this-host-does-not-exist.invalid"
}

data "streamkap_topics" "test" {}
`,
				// Should fail with connection error (config host used), not succeed (env host)
				ExpectError: MustCompile(`(?i)(unable|connect|network|dial|host|refused|no such host)`),
			},
		},
	})
}

// TestProviderConfig_DefaultHost tests that host defaults to production API if not set
func TestProviderConfig_DefaultHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Clear host environment variable to test default
	originalHost := os.Getenv("STREAMKAP_HOST")
	os.Unsetenv("STREAMKAP_HOST")
	defer func() {
		if originalHost != "" {
			os.Setenv("STREAMKAP_HOST", originalHost)
		}
	}()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "streamkap" {
  # host not set - should default to https://api.streamkap.com
  client_id = "invalid-client-id"
  secret    = "invalid-secret"
}

data "streamkap_topics" "test" {}
`,
				// Should fail with auth error (reaching real API with invalid creds)
				// This proves the default host was used
				ExpectError: MustCompile(`(?i)(unauthorized|invalid|credentials|authentication|forbidden|access)`),
			},
		},
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// MustCompile is a helper that wraps regexp.MustCompile for test error patterns
func MustCompile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
