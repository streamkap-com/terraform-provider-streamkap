package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTagResource_descriptionShapes pins the null-versus-empty handling of
// `description`. The backend stores both an omitted and an empty description
// as "", so the provider has to reproduce whichever shape the configuration
// used or every plan after apply shows a diff.
func TestAccTagResource_descriptionShapes(t *testing.T) {
	name := acctestName(t, "tag-description")
	config := func(descriptionLine string) string {
		return providerConfig + fmt.Sprintf(`
resource "streamkap_tag" "test" {
	name = %q
	type = ["sources"]
	%s
}
`, name, descriptionLine)
	}
	addr := "streamkap_tag.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTagDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(addr, "description"),
			},
			{
				Config: config(`description = ""`),
				Check:  resource.TestCheckResourceAttr(addr, "description", ""),
			},
			{
				Config: config(`description = "owned by terraform"`),
				Check:  resource.TestCheckResourceAttr(addr, "description", "owned by terraform"),
			},
			{
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(addr, "description"),
			},
		},
	})
}
