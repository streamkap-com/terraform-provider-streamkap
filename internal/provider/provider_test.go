package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	// ProviderConfig is a shared configuration to combine with the actual
	// test configuration so the HashiCups client is properly configured.
	// It is also possible to use the HASHICUPS_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	providerConfig = `
provider "streamkap" {}
`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"streamkap": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck is a helper function called by acceptance tests to verify
// required environment variables are set before running tests.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("STREAMKAP_CLIENT_ID"); v == "" {
		t.Fatal("STREAMKAP_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("STREAMKAP_SECRET"); v == "" {
		t.Fatal("STREAMKAP_SECRET must be set for acceptance tests")
	}
}

// legacyProviderConfig returns ExternalProvider config for the OLD provider (v2.1.18)
// Used in migration tests to create state with old provider, then verify new provider
// produces no planned changes.
//
// Requirements:
// - v2.1.18 must be available in Terraform Registry
// - If provider fetch fails, migration tests will error (not skip)
// - Verify with: terraform providers mirror -platform=linux_amd64 /tmp/mirror
//
// TEMPORARY: Delete this after v3.0.0 release is validated.
func legacyProviderConfig() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"streamkap": {
			VersionConstraint: "2.1.18",
			Source:            "streamkap-com/streamkap",
		},
	}
}

