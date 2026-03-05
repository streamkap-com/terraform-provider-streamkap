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
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
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
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func TestAccTransformMapFilterResource_deploy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with deploy = true
			{
				Config: testAccTransformMapFilterDeployConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy", "name", "tf-acc-test-transform-map-filter-deploy"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy", "deploy", "true"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_deploy", "connector_status"),
				),
			},
			// Step 2: Import
			{
				ResourceName:            "streamkap_transform_map_filter.test_deploy",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func TestAccTransformMapFilterResource_deployWithReplayWindow(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformMapFilterDeployWithReplayWindowConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy_replay", "name", "tf-acc-test-transform-map-filter-deploy-replay"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy_replay", "deploy", "true"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy_replay", "replay_window", "0"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_deploy_replay", "connector_status"),
				),
			},
			{
				ResourceName:            "streamkap_transform_map_filter.test_deploy_replay",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
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

func testAccTransformMapFilterDeployConfig() string {
	return `
resource "streamkap_transform_map_filter" "test_deploy" {
  name                                   = "tf-acc-test-transform-map-filter-deploy"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
  transforms_language                    = "JavaScript"

  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj) { return inputObj; }"
  })

  deploy = true
}
`
}

func testAccTransformMapFilterDeployWithReplayWindowConfig() string {
	return `
resource "streamkap_transform_map_filter" "test_deploy_replay" {
  name                                   = "tf-acc-test-transform-map-filter-deploy-replay"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
  transforms_language                    = "JavaScript"

  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj) { return inputObj; }"
  })

  deploy        = true
  replay_window = "0"
}
`
}
