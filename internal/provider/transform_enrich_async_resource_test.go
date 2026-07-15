package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformEnrichAsyncResource_basic(t *testing.T) {
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccTransformEnrichAsyncResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich_async.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich_async.test", "transform_type"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_timeout_ms", "1000"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_capacity", "10"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_input_topic_pattern", "test-input-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_output_topic_pattern", "test-output-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_input_serialization_format", "Avro"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_output_serialization_format", "Avro"),
				),
			},
			// ImportState
			{
				ResourceName:            "streamkap_transform_enrich_async.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
			// Update
			{
				Config: testAccTransformEnrichAsyncResourceConfigUpdated(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_language", "Python"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_timeout_ms", "2000"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_capacity", "20"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_input_serialization_format", "Json"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_output_serialization_format", "Json"),
				),
			},
		},
	})
}

func TestAccTransformEnrichAsyncResource_withImplementation(t *testing.T) {
	name := acctestName(t, "impl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create with implementation_json
			{
				Config: testAccTransformEnrichAsyncWithImplementationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test_impl", "name", name),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test_impl", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich_async.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich_async.test_impl", "implementation_json"),
				),
			},
			// ImportState
			{
				ResourceName:            "streamkap_transform_enrich_async.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func testAccTransformEnrichAsyncResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_enrich_async" "test" {
  name                                   = %q
  transforms_language                    = "JavaScript"
  transforms_async_timeout_ms            = 1000
  transforms_async_capacity              = 10
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`, name)
}

func testAccTransformEnrichAsyncResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_enrich_async" "test" {
  name                                   = %q
  transforms_language                    = "Python"
  transforms_async_timeout_ms            = 2000
  transforms_async_capacity              = 20
  transforms_input_topic_pattern         = "test-input-topic-updated"
  transforms_output_topic_pattern        = "test-output-topic-updated"
  transforms_input_serialization_format  = "Json"
  transforms_output_serialization_format = "Json"
}
`, name)
}

func testAccTransformEnrichAsyncWithImplementationConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_enrich_async" "test_impl" {
  name                                   = %q
  transforms_language                    = "JavaScript"
  transforms_async_timeout_ms            = 1000
  transforms_async_capacity              = 10
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"

  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj, keyObj, topicName, timestamp, commonObject) { return inputObj; }"
  })
}
`, name)
}
