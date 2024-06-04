package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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

func TestAccPreCheck(t *testing.T) {
	if v := os.Getenv("STREAMKAP_CLIENT_ID"); v == "" {
		t.Fatal("STREAMKAP_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("STREAMKAP_SECRET"); v == "" {
		t.Fatal("STREAMKAP_SECRET must be set for acceptance tests")
	}
}
