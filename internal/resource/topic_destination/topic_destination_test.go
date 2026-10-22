package topic_destination

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// bindingClient reads the binding's topic list and writes the whole list back
// without a lock, so it loses topics unless the provider serializes one
// source's changes.
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

func TestTopicDestinationSerializesChangesPerSource(t *testing.T) {
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
			"timeouts":       tftypes.NewValue(objType.AttributeTypes["timeouts"], nil),
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

// A topic ID the backend cannot resolve to an API source fails at apply; the plan catches its shape, including the
// "|" that would break the import ID.
func TestTopicDestinationTopicIDShape(t *testing.T) {
	var response resource.SchemaResponse
	(&Resource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	validators := response.Schema.Attributes["topic_id"].(schema.StringAttribute).Validators
	valid := func(topicID string) bool {
		var out validator.StringResponse
		for _, v := range validators {
			v.ValidateString(context.Background(), validator.StringRequest{
				Path: path.Root("topic_id"), ConfigValue: types.StringValue(topicID),
			}, &out)
		}
		return !out.Diagnostics.HasError()
	}
	for _, topicID := range []string{"source_abc.salesforce.Account", "source_abc.google_analytics.custom_daily.users"} {
		if !valid(topicID) {
			t.Errorf("refused %q", topicID)
		}
	}
	for _, topicID := range []string{"Account", "source_abc.Account", "abc.salesforce.Account", "source_abc.salesforce.Acc|ount", "source_.salesforce.Account"} {
		if valid(topicID) {
			t.Errorf("accepted %q", topicID)
		}
	}
}

// An import ID is checked like topic_id in a configuration, before Read asks the API for a topic that cannot exist.
func TestTopicDestinationImportValidatesTopicID(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&Resource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	importID := func(id string) resource.ImportStateResponse {
		objType := schemaResp.Schema.Type().TerraformType(ctx)
		resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
		(&Resource{}).ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
		return resp
	}
	resp := importID("foo|dest-1")
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "source_<source id>.<connector>.<resource>") {
		t.Fatalf("import of a malformed topic ID must be refused with its expected shape: %v", resp.Diagnostics)
	}
	if resp := importID("source_abc.hubspot.contacts|dest-1"); resp.Diagnostics.HasError() {
		t.Fatalf("a valid import ID was refused: %v", resp.Diagnostics)
	}
}

// A change queued behind others for the same source starts its timeout once it
// holds the source, so a long for_each still finishes; an interrupt ends the
// wait.
func TestTopicDestinationLockWaitAndTimeout(t *testing.T) {
	ctx := context.Background()
	r := &Resource{client: &bindingClient{topics: map[string][]string{}}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	timeoutsType := objType.AttributeTypes["timeouts"].(tftypes.Object)
	timeoutsValue := map[string]tftypes.Value{}
	for name := range timeoutsType.AttributeTypes {
		timeoutsValue[name] = tftypes.NewValue(tftypes.String, "50ms")
	}
	plan := func(topicID string) tfsdk.Plan {
		return tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, nil),
			"topic_id":       tftypes.NewValue(tftypes.String, topicID),
			"destination_id": tftypes.NewValue(tftypes.String, "dest-1"),
			"binding_id":     tftypes.NewValue(tftypes.String, nil),
			"timeouts":       tftypes.NewValue(timeoutsType, timeoutsValue),
		})}
	}
	create := func(ctx context.Context, topicID string) <-chan resource.CreateResponse {
		done := make(chan resource.CreateResponse, 1)
		go func() {
			resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan(topicID)}, &resp)
			done <- resp
		}()
		return done
	}

	unlock, err := lockSource(ctx, "source_busy.hubspot.deals")
	if err != nil {
		t.Fatal(err)
	}
	queued := create(ctx, "source_busy.hubspot.contacts")
	time.Sleep(150 * time.Millisecond) // three times the 50ms timeout
	unlock()
	select {
	case resp := <-queued:
		if resp.Diagnostics.HasError() {
			t.Fatalf("time queued behind another change used up the create timeout: %v", resp.Diagnostics)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("create never ran after the source was released")
	}

	unlock, err = lockSource(ctx, "source_held.hubspot.deals")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	interrupted, cancel := context.WithCancel(ctx)
	waiting := create(interrupted, "source_held.hubspot.contacts")
	cancel()
	select {
	case resp := <-waiting:
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "source_held") {
			t.Fatalf("an interrupted wait must fail naming the source: %v", resp.Diagnostics)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("an interrupt did not end the wait for the source")
	}
}

// Only timeouts reach Update, because topic_id and destination_id replace the
// resource; a changed timeout is stored without touching the binding.
func TestTopicDestinationUpdateStoresChangedTimeouts(t *testing.T) {
	ctx := context.Background()
	r := &Resource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	timeoutsType := objType.AttributeTypes["timeouts"].(tftypes.Object)
	value := func(create string) tftypes.Value {
		timeoutsValue := map[string]tftypes.Value{}
		for name := range timeoutsType.AttributeTypes {
			timeoutsValue[name] = tftypes.NewValue(tftypes.String, nil)
		}
		timeoutsValue["create"] = tftypes.NewValue(tftypes.String, create)
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, "source_1.hubspot.contacts|dest-1"),
			"topic_id":       tftypes.NewValue(tftypes.String, "source_1.hubspot.contacts"),
			"destination_id": tftypes.NewValue(tftypes.String, "dest-1"),
			"binding_id":     tftypes.NewValue(tftypes.String, "binding-1"),
			"timeouts":       tftypes.NewValue(timeoutsType, timeoutsValue),
		})
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: value("20m")}}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: value("40m")},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: value("20m")},
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a timeouts-only change must apply: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.Equal(value("40m")) {
		t.Fatalf("state kept the old timeouts: %v", resp.State.Raw)
	}
}
