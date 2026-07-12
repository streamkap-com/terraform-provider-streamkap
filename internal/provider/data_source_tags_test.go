package provider

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// uniqSuffix derives a short, alphanumeric suffix unique to this test
// invocation — used to keep parallel/repeated runs from colliding on the
// shared backend (which enforces unique tag names per tenant).
func uniqSuffix(t *testing.T) string {
	t.Helper()
	cleaned := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	cleaned = strings.ReplaceAll(cleaned, "_", "-")
	return fmt.Sprintf("%s-%d", cleaned, time.Now().UnixNano())
}

// acctestName builds a connector/pipeline fixture name for this test invocation.
//
// Two properties matter:
//   - the `tf-acc-test` prefix is what sweep_test.go's isTestResource matches, so
//     a leaked fixture is sweepable;
//   - the uniqSuffix keeps two runs (a PR run and the nightly, or a rerun racing
//     a leaked fixture) from colliding on {tenant, name}. The backend 422s on a
//     duplicate name, and sources/destinations no longer adopt the existing
//     record, so a collision now fails the apply outright.
//
// role distinguishes the fixtures of one test from each other ("source",
// "destination", "updated", ...).
func acctestName(t *testing.T, role string) string {
	t.Helper()
	return fmt.Sprintf("tf-acc-test-%s-%s", uniqSuffix(t), role)
}

// TestAccDataSourceTags_filterName creates two custom tags and asserts the
// streamkap_tags data source returns the expected one when filter_name is
// applied. Filter-by-name is the most common end-user flow ("look up the prod
// tag ID without hardcoding").
func TestAccDataSourceTags_filterName(t *testing.T) {
	uniq := uniqSuffix(t)
	tagAName := fmt.Sprintf("tf-acc-test-tags-ds-%s-A", uniq)
	tagBName := fmt.Sprintf("tf-acc-test-tags-ds-%s-B", uniq)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "a" {
	name        = %q
	description = "Tag A"
	type        = ["sources"]
}

resource "streamkap_tag" "b" {
	name        = %q
	description = "Tag B"
	type        = ["destinations"]
}

# Look up tag A by name. Two tags exist on the tenant matching different
# substrings; filter_name must match exactly to return only one.
data "streamkap_tags" "by_name" {
	filter_name = streamkap_tag.a.name
	depends_on  = [streamkap_tag.a, streamkap_tag.b]
}
`, tagAName, tagBName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Exactly one tag should match by exact name.
					resource.TestCheckResourceAttr("data.streamkap_tags.by_name", "tags.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.streamkap_tags.by_name", "tags.0.id",
						"streamkap_tag.a", "id"),
					resource.TestCheckResourceAttr("data.streamkap_tags.by_name", "tags.0.name", tagAName),
					resource.TestCheckResourceAttr("data.streamkap_tags.by_name", "tags.0.description", "Tag A"),
				),
			},
		},
	})
}

// TestAccDataSourceTags_filterIds verifies filter_ids returns exactly the
// matching records and that the rendered objects expose all schema fields.
func TestAccDataSourceTags_filterIds(t *testing.T) {
	uniq := uniqSuffix(t)
	tagName := fmt.Sprintf("tf-acc-test-tags-ds-ids-%s", uniq)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "subject" {
	name = %q
	type = ["pipelines", "sources"]
}

data "streamkap_tags" "by_id" {
	filter_ids = [streamkap_tag.subject.id]
}
`, tagName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.streamkap_tags.by_id", "tags.#", "1"),
					resource.TestCheckResourceAttrPair(
						"data.streamkap_tags.by_id", "tags.0.id",
						"streamkap_tag.subject", "id"),
					resource.TestCheckResourceAttr("data.streamkap_tags.by_id", "tags.0.name", tagName),
					resource.TestCheckResourceAttr("data.streamkap_tags.by_id", "tags.0.custom", "true"),
					resource.TestCheckResourceAttr("data.streamkap_tags.by_id", "tags.0.system", "false"),
					// type Set should round-trip both values.
					resource.TestCheckTypeSetElemAttr("data.streamkap_tags.by_id", "tags.0.type.*", "pipelines"),
					resource.TestCheckTypeSetElemAttr("data.streamkap_tags.by_id", "tags.0.type.*", "sources"),
				),
			},
		},
	})
}
