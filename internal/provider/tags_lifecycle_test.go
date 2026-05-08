package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTagsLifecycle_TransformAttachAndClear exercises the contract that
// the code-reviewer audit flagged: setting `tags = [...]` on an entity must
// attach the tags, and changing to `tags = []` must clear them. The bug
// before commit 14214ac was that Go's encoding/json with omitempty on a
// non-nil zero-length slice produced "field absent" on the wire — backend
// then treated it as "do not change", and `tags = []` silently no-op'd.
//
// We use a transform here because it is the cheapest non-pipeline connector
// to create against the real API. Sources/destinations need real database
// hostnames; transforms only need a name.
func TestAccTagsLifecycle_TransformAttachAndClear(t *testing.T) {
	uniq := uniqSuffix(t)
	tagName := fmt.Sprintf("tf-acc-test-tags-lifecycle-%s", uniq)
	transformName := fmt.Sprintf("tf-acc-test-tx-tags-%s", uniq)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Step 1: create a transform with one tag attached.
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "lifecycle" {
	name = %q
	type = ["transforms"]
}

resource "streamkap_transform_map_filter" "test" {
	name = %q
	tags = [streamkap_tag.lifecycle.id]
}
`, tagName, transformName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "tags.#", "1"),
					resource.TestCheckResourceAttrPair(
						"streamkap_transform_map_filter.test", "tags.0",
						"streamkap_tag.lifecycle", "id"),
				),
			},
			// Step 2: change to `tags = []` — must clear, not no-op. This is
			// the regression guard for the omitempty bug.
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "lifecycle" {
	name = %q
	type = ["transforms"]
}

resource "streamkap_transform_map_filter" "test" {
	name = %q
	tags = []
}
`, tagName, transformName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "tags.#", "0"),
				),
			},
			// Step 3: re-attach a (different) tag — proves Update can grow
			// from empty back to non-empty.
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "lifecycle" {
	name = %q
	type = ["transforms"]
}

resource "streamkap_transform_map_filter" "test" {
	name = %q
	tags = [streamkap_tag.lifecycle.id]
}
`, tagName, transformName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "tags.#", "1"),
				),
			},
		},
	})
}
