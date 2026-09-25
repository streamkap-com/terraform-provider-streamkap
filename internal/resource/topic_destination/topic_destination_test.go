package topic_destination

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestTopicDestinationIdentity(t *testing.T) {
	for _, id := range []string{"", "topic", "|dest", "topic|", "topic|dest|other"} {
		if _, _, err := parseCompositeID(id); err == nil {
			t.Fatalf("accepted invalid import ID %q", id)
		}
	}
	topicID, destinationID, err := parseCompositeID(compositeID("source_1.Account", "dest-1"))
	if err != nil || topicID != "source_1.Account" || destinationID != "dest-1" {
		t.Fatalf("parsed %q, %q, %v", topicID, destinationID, err)
	}
	var response resource.SchemaResponse
	(&Resource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	for _, name := range []string{"topic_id", "destination_id"} {
		field := response.Schema.Attributes[name].(schema.StringAttribute)
		if !field.Required || len(field.PlanModifiers) == 0 {
			t.Fatalf("%s must require replacement", name)
		}
	}
}
