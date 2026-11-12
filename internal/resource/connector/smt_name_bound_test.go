package connector_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

// The chain write bounds an instance's display name at 200 characters, like
// its minimum of one, so a longer name fails the plan on smt_chain[0].name
// and nothing is sent.
func TestSMT_InstanceNameBound(t *testing.T) {
	factories, fake := newFixture(t)
	named := func(name string) string {
		return strings.Replace(chainAB, `name = "East"`, fmt.Sprintf("name = %q", name), 1)
	}
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config:      hcl(named(strings.Repeat("é", 201))),
				ExpectError: regexp.MustCompile(`smt_chain\[0\]\.name[\s\S]*between 1 and 200`),
			},
			{
				PreConfig: func() { require.Zero(t, fake.writeCount(), "a refused name must send nothing") },
				Config:    hcl(named(strings.Repeat("é", 200))),
			},
		},
	})
}
