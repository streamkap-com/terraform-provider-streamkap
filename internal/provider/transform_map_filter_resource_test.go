package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformMapFilterResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformMapFilterResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "name", "tf-acc-test-transform-map-filter"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "transforms_input_serialization_format", "Avro"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "transforms_output_serialization_format", "Avro"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test", "transform_type"),
				),
			},
			{
				ResourceName:            "streamkap_transform_map_filter.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
		},
	})
}

func TestAccTransformMapFilterResource_withImplementation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with implementation_json
			{
				Config: testAccTransformMapFilterWithImplementationConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_impl", "name", "tf-acc-test-transform-map-filter-impl"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_impl", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_impl", "implementation_json"),
				),
			},
			// Step 2: Update implementation_json
			{
				Config: testAccTransformMapFilterWithImplementationUpdatedConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_impl", "name", "tf-acc-test-transform-map-filter-impl"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_impl", "implementation_json"),
				),
			},
			// Step 3: Import
			{
				ResourceName:            "streamkap_transform_map_filter.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
		},
	})
}

func testAccTransformMapFilterResourceConfig() string {
	return `
resource "streamkap_transform_map_filter" "test" {
  name                                   = "tf-acc-test-transform-map-filter"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
  transforms_language                    = "Python"
}
`
}

func testAccTransformMapFilterWithImplementationConfig() string {
	return `
resource "streamkap_transform_map_filter" "test_impl" {
  name                                   = "tf-acc-test-transform-map-filter-impl"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
  transforms_language                    = "JavaScript"

  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj) { return inputObj; }"
  })
}
`
}

func testAccTransformMapFilterWithImplementationUpdatedConfig() string {
	return `
resource "streamkap_transform_map_filter" "test_impl" {
  name                                   = "tf-acc-test-transform-map-filter-impl"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
  transforms_language                    = "JavaScript"

  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj) { inputObj.processed = true; return inputObj; }"
  })
}
`
}
