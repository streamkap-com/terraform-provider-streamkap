package tag

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func TestModelFromAPIObjectPreservesEmptyDescriptionShape(t *testing.T) {
	tests := map[string]struct {
		priorDescription types.String
		wantNull         bool
	}{
		"omitted": {
			priorDescription: types.StringNull(),
			wantNull:         true,
		},
		"explicit empty": {
			priorDescription: types.StringValue(""),
			wantNull:         false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			model := TagResourceModel{Description: test.priorDescription}
			resource := TagResource{}
			resource.modelFromAPIObject(api.Tag{ID: "tag-id", Name: "tag", Description: ""}, &model)

			if model.Description.IsNull() != test.wantNull {
				t.Fatalf("description null = %t, want %t", model.Description.IsNull(), test.wantNull)
			}
		})
	}
}
