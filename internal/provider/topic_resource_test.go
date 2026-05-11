package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTopicResource exercises create → update → delete on streamkap_topic.
//
// This test is currently self-poisoning: Step 1 creates the topic with 25
// partitions, Step 3 updates to 26. Kafka partition counts can only
// increase, so the second run sees the topic at 26 already and Step 1
// (which sets 25) fails with "New partition count (25) must be greater
// than the current count (26)".
//
// The backend's DELETE /topics/{id} only removes the streamkap DB record;
// the underlying Kafka topic survives at whatever count the previous run
// left it. We confirmed this by adding a PreConfig that calls
// client.DeleteTopic — the streamkap-side delete returns "Could not find
// the topic" while the Kafka topic still reports 26 partitions on the
// next streamkap_topic create.
//
// Fixing this properly requires either:
//   - direct Kafka admin access from the test (out of scope; tests run
//     against api.streamkap.com via HTTPS only), or
//   - reworking the test to discover the live partition count and write
//     Step 1/3 configs dynamically (Config is static HCL via
//     terraform-plugin-testing, so this would need PreConfig + a
//     templated config string).
//
// Until the backend exposes a way to reset a topic's Kafka partition
// count, the test must be skipped to avoid a perpetual red signal that
// has nothing to do with provider code.
func TestAccTopicResource(t *testing.T) {
	t.Skip("self-poisoning: Kafka partition counts can only increase, " +
		"so each run leaves the fixture topic in a state that breaks " +
		"the next run's Step 1 (see comment header for the fix paths)")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTopicDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing
			{
				Config: providerConfig + `
resource "streamkap_topic" "test" {
	topic_id                                   = "source_67adbcc172417ef6338e01a1.default.tst-junit-2"
  	partition_count                            = 25
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_topic.test", "topic_id", "source_67adbcc172417ef6338e01a1.default.tst-junit-2"),
					resource.TestCheckResourceAttr("streamkap_topic.test", "partition_count", "25"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "topic_id.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing
			{
				Config: providerConfig + `
resource "streamkap_topic" "test" {
	topic_id                                   = "source_67adbcc172417ef6338e01a1.default.tst-junit-2"
  	partition_count                            = 26
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_topic.test", "topic_id", "source_67adbcc172417ef6338e01a1.default.tst-junit-2"),
					resource.TestCheckResourceAttr("streamkap_topic.test", "partition_count", "26"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
