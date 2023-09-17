package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"streamkap": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccPreCheck(t *testing.T) {
	if v := os.Getenv("STREAMKAP_HOST"); v == "" {
		t.Fatal("STREAMKAP_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("STREAMKAP_CLIENT_ID"); v == "" {
		t.Fatal("STREAMKAP_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("STREAMKAP_SECRET"); v == "" {
		t.Fatal("STREAMKAP_SECRET must be set for acceptance tests")
	}
}
