package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/provider"
)

//nolint:lll
func TestAccTokenDataSource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", "https://api.streamkap.com/api/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			token := make(map[string]interface{})
			token["accessToken"] = "jwtToken"
			token["refreshToken"] = "refresh-token"
			token["expiresIn"] = 3600
			token["expires"] = "Fri, 15 Sep 2023 16:48:11 GMT"

			if err := json.NewDecoder(req.Body).Decode(&token); err != nil {
				fmt.Println("Error decode", err)
				return httpmock.NewStringResponse(400, ""), nil
			}

			resp, err := httpmock.NewJsonResponse(200, token)
			if err != nil {
				fmt.Println("Error", err)
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.streamkap_token.test", "access_token", "jwtToken"),
				),
			},
		},
	})
}

const testAccTokenDataSourceConfig = `
data "streamkap_token" "test" {}
`
