package provider

//func TestAccPipelineResource(t *testing.T) {
//	httpmock.Activate()
//	defer httpmock.DeactivateAndReset()
//	httpmock.RegisterResponder("POST", "https://api.streamkap.com/api/auth/access-token",
//		func(req *http.Request) (*http.Response, error) {
//			token := make(map[string]interface{})
//			token["accessToken"] = "jwtToken"
//			token["refreshToken"] = "refresh-token"
//			token["expiresIn"] = 3600
//			token["expires"] = "Fri, 15 Sep 2023 16:48:11 GMT"
//
//			if err := json.NewDecoder(req.Body).Decode(&token); err != nil {
//				return httpmock.NewStringResponse(400, ""), nil
//			}
//
//			resp, err := httpmock.NewJsonResponse(200, token)
//			if err != nil {
//				return httpmock.NewStringResponse(500, ""), nil
//			}
//			return resp, nil
//		})
//
//	httpmock.RegisterResponder("POST", "https://api.streamkap.com/api/pipelines",
//		func(req *http.Request) (*http.Response, error) {
//			source := &api.Pipeline{
//				ID:   "example-id",
//				Name: "one",
//			}
//
//			resp, err := httpmock.NewJsonResponse(200, source)
//			if err != nil {
//				return httpmock.NewStringResponse(500, ""), nil
//			}
//			return resp, nil
//		})
//	httpmock.RegisterResponder("GET", "https://api.streamkap.com/api/pipelines/example-id",
//		func(req *http.Request) (*http.Response, error) {
//			source := []api.Pipeline{
//				{
//					ID:   "example-id",
//					Name: "one",
//				},
//			}
//
//			resp, err := httpmock.NewJsonResponse(200, source)
//			if err != nil {
//				return httpmock.NewStringResponse(500, ""), nil
//			}
//			return resp, nil
//		})
//	httpmock.RegisterResponder("GET", "https://api.streamkap.com/api/pipelines/example-id",
//		func(req *http.Request) (*http.Response, error) {
//			source := []api.Pipeline{
//				{
//					ID:   "example-id",
//					Name: "one",
//				},
//			}
//			resp, err := httpmock.NewJsonResponse(200, source)
//			if err != nil {
//				return httpmock.NewStringResponse(500, ""), nil
//			}
//			return resp, nil
//		})
//
//	resource.Test(t, resource.TestCase{
//		PreCheck:                 func() { TestAccPreCheck(t) },
//		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
//		Steps: []resource.TestStep{
//			// Create and Read testing
//			{
//				Config: testAccPipelineResourceConfig("one"),
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckResourceAttr("streamkap_pipeline.test", "name", "one"),
//					resource.TestCheckResourceAttr("streamkap_pipeline.test", "connector", "mysql"),
//					resource.TestCheckResourceAttr("streamkap_pipeline.test", "id", "example-id"),
//				),
//			},
//			// ImportState testing
//			{
//				ResourceName:      "streamkap_pipeline.test",
//				ImportState:       true,
//				ImportStateVerify: false,
//				ImportStateId:     "example-id",
//			},
//			// Update and Read testing
//			{
//				Config: testAccPipelineResourceConfig("two"),
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckResourceAttr("streamkap_pipeline.test", "name", "two"),
//				),
//			},
//			// Delete testing automatically occurs in TestCase
//		},
//	})
//}
//
//func testAccPipelineResourceConfig(configurableAttribute string) string {
//	return fmt.Sprintf(`
//resource "streamkap_pipeline" "test" {
//	name = %q
//	connector = "mysql"
//	config = jsonencode({
//		"database.hostname.user.defined"= "192.168.3.47"
//		"database.port"= "3306"
//		"database.user"= "root"
//		"database.password"= "iAxki9j9fr8H8LV"
//		"database.include.list.user.defined"= "database1, database2"
//		"table.include.list.user.defined"= "database1.table1, database1.table2, database2.table3, database2.table4"
//		"signal.data.collection.schema.or.database"= "test1"
//		"database.connectionTimeZone"= "SERVER"
//		"snapshot.gtid"= "No"
//		"snapshot.mode.user.defined"= "When Needed"
//		"binary.handling.mode"= "bytes"
//		"incremental.snapshot.chunk.size"= 1024
//		"max.batch.size"= 2048
//	})
//}
//`, configurableAttribute)
//}
