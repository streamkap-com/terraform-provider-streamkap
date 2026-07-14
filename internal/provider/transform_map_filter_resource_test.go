package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformMapFilterResource_basic(t *testing.T) {
	name := acctestName(t, "main")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformMapFilterResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "name", name),
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
	name := acctestName(t, "impl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with implementation_json
			{
				Config: testAccTransformMapFilterWithImplementationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_impl", "name", name),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_impl", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test_impl", "implementation_json"),
				),
			},
			// Step 2: Update implementation_json
			{
				Config: testAccTransformMapFilterWithImplementationUpdatedConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_impl", "name", name),
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
	name := acctestName(t, "deploy")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with deploy = true
			{
				Config: testAccTransformMapFilterDeployConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy", "name", name),
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
	name := acctestName(t, "deploy-replay")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformMapFilterDeployWithReplayWindowConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test_deploy_replay", "name", name),
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

func testAccTransformMapFilterResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_map_filter" "test" {
  name                                   = %q
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
  transforms_language                    = "Python"
}
`, name)
}

func testAccTransformMapFilterWithImplementationConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_map_filter" "test_impl" {
  name                                   = %q
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
`, name)
}

func testAccTransformMapFilterWithImplementationUpdatedConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_map_filter" "test_impl" {
  name                                   = %q
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
`, name)
}

func testAccTransformMapFilterDeployConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_map_filter" "test_deploy" {
  name                                   = %q
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
`, name)
}

func testAccTransformMapFilterDeployWithReplayWindowConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_map_filter" "test_deploy_replay" {
  name                                   = %q
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
`, name)
}
