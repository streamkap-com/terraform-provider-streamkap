package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/provider"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// Every Sensitive connector attribute travels under its API field name in a
// logged request body, so the redaction pattern must match each of them.
func TestRedactSensitiveJSON_CoversEverySensitiveConnectorField(t *testing.T) {
	ctx := context.Background()
	for _, factory := range provider.New("test")().Resources(ctx) {
		base, ok := factory().(*connector.BaseConnectorResource)
		if !ok {
			continue
		}
		config := base.Config()
		mappings := config.GetFieldMappings()
		for name, attribute := range config.GetSchema().Attributes {
			apiField, mapped := mappings[name]
			if !attribute.IsSensitive() || !mapped {
				continue
			}
			logged := api.RedactSensitiveJSON([]byte(`{"` + apiField + `":"leaked-value"}`))
			if strings.Contains(logged, "leaked-value") {
				t.Errorf("%s %s: API field %q is Sensitive but not redacted from request logs", config.GetConnectorType(), config.GetConnectorCode(), apiField)
			}
		}
	}
}
