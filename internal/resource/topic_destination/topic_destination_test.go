package topic_destination

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
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

// bindingClient mimics the backend: attach and detach read the binding's topic
// list, then write the whole list back, with no lock between the two.
type bindingClient struct {
	api.StreamkapAPI
	mu     sync.Mutex
	topics map[string][]string
}

func (c *bindingClient) rewrite(destinationID string, change func([]string) []string) []string {
	c.mu.Lock()
	current := append([]string(nil), c.topics[destinationID]...)
	c.mu.Unlock()
	time.Sleep(5 * time.Millisecond)
	next := change(current)
	c.mu.Lock()
	c.topics[destinationID] = next
	c.mu.Unlock()
	return next
}

func (c *bindingClient) AttachTopicDestination(_ context.Context, topicID, destinationID string) (*api.TopicDestinationLink, error) {
	topics := c.rewrite(destinationID, func(current []string) []string {
		for _, topic := range current {
			if topic == topicID {
				return current
			}
		}
		return append(current, topicID)
	})
	return &api.TopicDestinationLink{BindingID: "binding-1", DestinationID: destinationID, TopicIDs: topics}, nil
}

func (c *bindingClient) DetachTopicDestination(_ context.Context, topicID, destinationID string) error {
	c.rewrite(destinationID, func(current []string) []string {
		remaining := current[:0]
		for _, topic := range current {
			if topic != topicID {
				remaining = append(remaining, topic)
			}
		}
		return remaining
	})
	return nil
}

func TestTopicDestinationSerializesChangesPerDestination(t *testing.T) {
	ctx := context.Background()
	client := &bindingClient{topics: map[string][]string{}}
	r := &Resource{client: client}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	value := func(topicID string) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, nil),
			"topic_id":       tftypes.NewValue(tftypes.String, topicID),
			"destination_id": tftypes.NewValue(tftypes.String, "dest-1"),
			"binding_id":     tftypes.NewValue(tftypes.String, nil),
		})
	}
	run := func(op func(topicID string)) {
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(topicID string) {
				defer wg.Done()
				op(topicID)
			}(fmt.Sprintf("source_1.hubspot.object%d", i))
		}
		wg.Wait()
	}

	run(func(topicID string) {
		resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: value(topicID)}}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("create %s: %v", topicID, resp.Diagnostics)
		}
	})
	if got := len(client.topics["dest-1"]); got != 8 {
		t.Fatalf("parallel attaches kept %d of 8 topics: %v", got, client.topics["dest-1"])
	}

	run(func(topicID string) {
		resp := resource.DeleteResponse{}
		r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: value(topicID)}}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("delete %s: %v", topicID, resp.Diagnostics)
		}
	})
	if got := client.topics["dest-1"]; len(got) != 0 {
		t.Fatalf("parallel detaches left %v", got)
	}
}
