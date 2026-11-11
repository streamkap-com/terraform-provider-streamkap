package connector_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

const importAddr = "streamkap_source_postgresql.u"

// importHCL configures addr, which creates the connector a released provider
// would have created, and importAddr, which the steps import a copy of it into.
var importHCL = hcl("") + strings.Replace(hcl(""), `"t"`, `"u"`, 1)

// releasedCopy stores a copy of the connector at addr, the way the backend
// holds one the released provider created, and returns its id.
func releasedCopy(t *testing.T, fake *fakeAPI, createdID string) string {
	t.Helper()
	created, err := fake.CreateSource(context.Background(), *fake.sources[createdID])
	require.NoError(t, err)
	return created.ID
}

// noChainImported checks an imported state has no chain attributes and holds
// the flat attributes the API returned, as the released provider's import did.
func noChainImported(states []*terraform.InstanceState) error {
	a := states[0].Attributes
	for _, name := range []string{"smt_chain.#", "smt_predicates.#", "smt_managed_overrides.%"} {
		if _, has := a[name]; has {
			return fmt.Errorf("import set %s = %q", name, a[name])
		}
	}
	if a["smt_chain_revision"] != "" {
		return fmt.Errorf("import set smt_chain_revision = %q", a["smt_chain_revision"])
	}
	for name, value := range a {
		if strings.HasPrefix(name, "transforms_") && value != "" {
			return nil
		}
	}
	return fmt.Errorf("import left the flat SMT attributes unread")
}

// A connector imported by its id alone is imported the way the released
// provider imports it: the chain is not read, the flat attributes come from the
// API, and a configuration without SMT attributes plans no change.
func TestSMT_ImportByIDMatchesReleasedProvider(t *testing.T) {
	factories, fake := newFixture(t)
	var createdID, importedID string
	var readsBefore int
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: hcl(""),
				Check:  func(s *terraform.State) error { createdID = attrOf(s, "id"); return nil },
			},
			{
				PreConfig: func() {
					importedID = releasedCopy(t, fake, createdID)
					readsBefore = fake.chainReadCount()
				},
				Config:             importHCL,
				ResourceName:       importAddr,
				ImportState:        true,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return importedID, nil },
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if reads := fake.chainReadCount() - readsBefore; reads != 0 {
						return fmt.Errorf("import read the chain %d times", reads)
					}
					return noChainImported(states)
				},
			},
			{Config: importHCL, PlanOnly: true},
		},
	})
}

// Importing with the chain suffix on a backend without the chain route
// imports the connector as the released provider does; any other chain read
// failure still fails the import.
func TestSMT_ChainImportWithoutChainRoute(t *testing.T) {
	factories, fake := newFixture(t)
	var createdID, importedID string
	chainImportID := func(*terraform.State) (string, error) { return importedID + "/smt_chain", nil }
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: hcl(""),
				Check:  func(s *terraform.State) error { createdID = attrOf(s, "id"); return nil },
			},
			{
				PreConfig: func() {
					importedID = releasedCopy(t, fake, createdID)
					fake.chainReadErr = &api.APIError{StatusCode: http.StatusServiceUnavailable, Detail: "unavailable"}
				},
				Config:            importHCL,
				ResourceName:      importAddr,
				ImportState:       true,
				ImportStateIdFunc: chainImportID,
				ExpectError:       regexp.MustCompile(`Error reading transform chain`),
			},
			{
				PreConfig:          func() { fake.chainReadErr = &api.APIError{StatusCode: http.StatusNotFound, Detail: "Not Found"} },
				Config:             importHCL,
				ResourceName:       importAddr,
				ImportState:        true,
				ImportStateIdFunc:  chainImportID,
				ImportStatePersist: true,
				ImportStateCheck:   noChainImported,
			},
			{Config: importHCL, PlanOnly: true},
		},
	})
}
