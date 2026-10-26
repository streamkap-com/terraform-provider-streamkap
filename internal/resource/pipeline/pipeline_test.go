package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// backendPrettyTopicName mirrors the backend's get_pretty_topic_name_from_id
// from python-be-streamkap/app/utils/fetch_utils.py: it strips everything
// before (and including) the first "." in the stored topic_id. Any fix for
// issue #78 must make the round-trip survive this reconstruction.
func backendPrettyTopicName(topicID string) string {
	parts := strings.SplitN(topicID, ".", 2)
	if len(parts) < 2 {
		return topicID
	}
	return parts[1]
}

// simulateBackendRoundTrip mimics what the backend does between Create and Read:
// it persists the outgoing PipelineTransform entries and reconstructs
// transform.topic from topic_id via get_pretty_topic_name_from_id.
func simulateBackendRoundTrip(out []*api.PipelineTransform) []*api.PipelineTransform {
	round := make([]*api.PipelineTransform, len(out))
	for i, pt := range out {
		pretty := backendPrettyTopicName(pt.TopicID)
		round[i] = &api.PipelineTransform{
			ID:        pt.ID,
			Name:      pt.Name,
			StartTime: pt.StartTime,
			Topic:     pretty,
			TopicID:   pt.TopicID,
		}
	}
	return round
}

func setOfStrings(t *testing.T, values ...string) types.Set {
	t.Helper()
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	s, diags := types.SetValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("failed to build set: %s", diags)
	}
	return s
}

// fakeTransformClient stubs the GetTransform call so the model→API step can
// resolve a transform's own Topics/TopicIDs without hitting HTTP.
type fakeTransformClient struct {
	api.StreamkapAPI
	transform *api.Transform
}

func (f *fakeTransformClient) GetTransform(_ context.Context, id string) (*api.Transform, error) {
	if f.transform == nil || f.transform.ID != id {
		return &api.Transform{ID: id}, nil
	}
	return f.transform, nil
}

// TestPipelineTransforms_RoundTrip_Issue78 asserts the full round-trip
// (model → API request → backend denormalization → API response → model)
// preserves every user-supplied topic value, including topic names with dots
// and hyphens like the one reported in issue #78.
func TestPipelineTransforms_RoundTrip_Issue78(t *testing.T) {
	tests := []struct {
		name            string
		transformTopics []string // what transform.Topics reports (simulating a newly-configured transform)
		transformIDs    []string // parallel topic_ids from the transform API
		sourceID        string
		userTopics      []string // what the user wrote in transforms[].topics
	}{
		{
			name:            "issue_78_dotted_hyphenated_topic_with_empty_transform_topics",
			transformTopics: nil,
			transformIDs:    nil,
			sourceID:        "66b0f1c9a1b2c3d4e5f60000",
			userTopics:      []string{"aleks_live.SiteAlert-us-Local"},
		},
		{
			name:            "multiple_topics_with_mixed_case_and_special_chars",
			transformTopics: nil,
			transformIDs:    nil,
			sourceID:        "66b0f1c9a1b2c3d4e5f60001",
			userTopics:      []string{"svc.Users", "svc.Order-Events", "svc.Payment_Details"},
		},
		{
			name:            "topic_already_known_to_transform_uses_backend_topic_id",
			transformTopics: []string{"public.users"},
			transformIDs:    []string{"source_66b0f1c9a1b2c3d4e5f60002.public.users"},
			sourceID:        "66b0f1c9a1b2c3d4e5f60002",
			userTopics:      []string{"public.users"},
		},
		// Reproduces the failure reported on v3 where deployed transforms have
		// topic_ids prefixed with "transform_<id>_<version>." (set by the backend
		// after Flink/topic-router deployment, see python-be-streamkap
		// api_transforms_utils.py:427). Users still write the pretty topic name
		// in their TF config; the round-trip must return the pretty name, not
		// the full transform-prefixed topic_id.
		{
			name: "deployed_transform_with_transform_prefixed_topic_ids_returns_pretty_names",
			transformTopics: []string{
				"aleks_commons.CoreDataConflict-Global",
				"aleks_commons.PollenData-Global",
			},
			transformIDs: []string{
				"transform_66b0f1c9a1b2c3d4e5f6aaaa_0.aleks_commons.CoreDataConflict-Global",
				"transform_66b0f1c9a1b2c3d4e5f6aaaa_0.aleks_commons.PollenData-Global",
			},
			sourceID: "66b0f1c9a1b2c3d4e5f60003",
			userTopics: []string{
				"aleks_commons.CoreDataConflict-Global",
				"aleks_commons.PollenData-Global",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			transformID := "66b0f1c9a1b2c3d4e5f6aaaa"
			r := &PipelineResource{
				client: &fakeTransformClient{
					transform: &api.Transform{
						ID:       transformID,
						Name:     "t1",
						Topics:   tc.transformTopics,
						TopicIDs: tc.transformIDs,
					},
				},
			}

			modelIn := []*PipelineTransformModel{
				{
					ID:     types.StringValue(transformID),
					Topics: setOfStrings(t, tc.userTopics...),
				},
			}

			apiOut, err := r.model2APITransforms(context.Background(), modelIn, tc.sourceID)
			if err != nil {
				t.Fatalf("model2APITransforms: %v", err)
			}
			if len(apiOut) != len(tc.userTopics) {
				t.Fatalf("expected %d unwound entries, got %d", len(tc.userTopics), len(apiOut))
			}

			// Every outgoing entry must survive get_pretty_topic_name_from_id:
			// everything after the first "." of topic_id must equal the
			// original user-supplied topic.
			for _, pt := range apiOut {
				pretty := backendPrettyTopicName(pt.TopicID)
				if pretty != pt.Topic {
					t.Fatalf("topic_id %q reconstructs to %q but Topic field is %q — backend will return %q, breaking round-trip",
						pt.TopicID, pretty, pt.Topic, pretty)
				}
			}

			// Now simulate the response as the backend would build it, and run
			// api2ModelTransforms over it. The resulting set must equal the
			// user's input set.
			apiResp := simulateBackendRoundTrip(apiOut)
			modelBack, err := r.api2ModelTransforms(context.Background(), apiResp)
			if err != nil {
				t.Fatalf("api2ModelTransforms: %v", err)
			}
			if len(modelBack) != 1 {
				t.Fatalf("expected 1 transform in round-trip model, got %d", len(modelBack))
			}

			gotTopics := []string{}
			if diags := modelBack[0].Topics.ElementsAs(context.Background(), &gotTopics, false); diags.HasError() {
				t.Fatalf("ElementsAs: %s", diags)
			}

			want := map[string]struct{}{}
			for _, s := range tc.userTopics {
				want[s] = struct{}{}
			}
			got := map[string]struct{}{}
			for _, s := range gotTopics {
				got[s] = struct{}{}
			}
			if len(want) != len(got) {
				t.Fatalf("round-trip topic count mismatch: input=%v roundtrip=%v", tc.userTopics, gotTopics)
			}
			for k := range want {
				if _, ok := got[k]; !ok {
					t.Fatalf("round-trip lost topic %q: input=%v roundtrip=%v", k, tc.userTopics, gotTopics)
				}
			}
		})
	}
}

// TestNormalizeSourceTopic asserts the source-topic prettifier strips only the
// exact "source_<id>." prefix and never corrupts already-pretty names — in
// particular dotted names like "default.MyTable" must survive intact.
func TestNormalizeSourceTopic(t *testing.T) {
	cases := []struct {
		name     string
		topic    string
		sourceID string
		want     string
	}{
		{"strips source prefix from bare name", "source_66b0.CoreData-Global", "66b0", "CoreData-Global"},
		{"strips prefix preserving dotted name", "source_66b0.default.MyTable", "66b0", "default.MyTable"},
		{"already pretty dotted name unchanged", "default.MyTable", "66b0", "default.MyTable"},
		{"already pretty bare name unchanged", "CoreData-Global", "66b0", "CoreData-Global"},
		{"empty sourceID returns topic verbatim", "source_66b0.X", "", "source_66b0.X"},
		{"non-matching source prefix unchanged", "source_OTHER.X", "66b0", "source_OTHER.X"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeSourceTopic(tc.topic, tc.sourceID); got != tc.want {
				t.Fatalf("normalizeSourceTopic(%q, %q) = %q, want %q", tc.topic, tc.sourceID, got, tc.want)
			}
		})
	}
}

// TestPipelineSourceTopics_RoundTrip_StripsPrefix asserts api2Model returns the
// pretty source topics even when the API echoes the raw "source_<id>.<name>"
// topic_id form (older betas / create-response path) — the source-topics
// analogue of the issue #78 transforms fix.
func TestPipelineSourceTopics_RoundTrip_StripsPrefix(t *testing.T) {
	r := &PipelineResource{}
	apiObj := api.Pipeline{
		ID:   "p1",
		Name: "p",
		Source: api.PipelineSource{
			ID:     "66b0f1c9a1b2c3d4e5f60000",
			Topics: []string{"source_66b0f1c9a1b2c3d4e5f60000.default.CoreData-Global", "default.PollenData"},
		},
	}

	model := &PipelineResourceModel{}
	if err := r.api2Model(context.Background(), apiObj, model); err != nil {
		t.Fatalf("api2Model: %v", err)
	}

	got := []string{}
	if diags := model.Source.Topics.ElementsAs(context.Background(), &got, false); diags.HasError() {
		t.Fatalf("ElementsAs: %s", diags)
	}

	want := map[string]struct{}{"default.CoreData-Global": {}, "default.PollenData": {}}
	if len(got) != len(want) {
		t.Fatalf("source topics = %v, want %v", got, want)
	}
	for _, g := range got {
		if _, ok := want[g]; !ok {
			t.Fatalf("unexpected source topic %q (got %v, want %v)", g, got, want)
		}
	}
}

// TestPipelineAutoDiscovery_RoundTrip asserts topic_auto_discovery_transforms
// is a clean pass-through: model → API request → API response → model
// preserves every {transform_id, regex} entry in order.
func TestPipelineAutoDiscovery_RoundTrip(t *testing.T) {
	r := &PipelineResource{client: &fakeTransformClient{}}

	in := []*PipelineTopicAutoDiscoveryModel{
		{TransformID: types.StringValue("66b0f1c9a1b2c3d4e5f6aaaa"), Regex: types.StringValue("aleks_ap.*")},
		{TransformID: types.StringValue("66b0f1c9a1b2c3d4e5f6bbbb"), Regex: types.StringValue("aleks_eu.*")},
	}

	model := PipelineResourceModel{
		Name:                         types.StringValue("p"),
		SnapshotNewTables:            types.BoolValue(true),
		Source:                       &PipelineSourceModel{ID: types.StringValue("66b0f1c9a1b2c3d4e5f60000"), Name: types.StringValue("s"), Connector: types.StringValue("dynamodb"), Topics: setOfStrings(t)},
		Destination:                  &PipelineDestinationModel{ID: types.StringValue("d1"), Name: types.StringValue("d"), Connector: types.StringValue("postgresql")},
		Tags:                         setOfStrings(t),
		TopicAutoDiscoveryTransforms: in,
	}

	apiObj, err := r.model2API(context.Background(), model)
	if err != nil {
		t.Fatalf("model2API: %v", err)
	}
	if len(apiObj.TopicAutoDiscoveryTransforms) != 2 {
		t.Fatalf("expected 2 auto-discovery entries on the API payload, got %d", len(apiObj.TopicAutoDiscoveryTransforms))
	}
	if apiObj.TopicAutoDiscoveryTransforms[0].TransformID != "66b0f1c9a1b2c3d4e5f6aaaa" ||
		apiObj.TopicAutoDiscoveryTransforms[0].Regex != "aleks_ap.*" {
		t.Fatalf("first entry mismatch: %+v", apiObj.TopicAutoDiscoveryTransforms[0])
	}

	back := &PipelineResourceModel{}
	if err := r.api2Model(context.Background(), *apiObj, back); err != nil {
		t.Fatalf("api2Model: %v", err)
	}
	if len(back.TopicAutoDiscoveryTransforms) != len(in) {
		t.Fatalf("round-trip count mismatch: got %d, want %d", len(back.TopicAutoDiscoveryTransforms), len(in))
	}
	for i, want := range in {
		got := back.TopicAutoDiscoveryTransforms[i]
		if got.TransformID.ValueString() != want.TransformID.ValueString() || got.Regex.ValueString() != want.Regex.ValueString() {
			t.Fatalf("round-trip entry %d mismatch: got {%s,%s} want {%s,%s}", i,
				got.TransformID.ValueString(), got.Regex.ValueString(),
				want.TransformID.ValueString(), want.Regex.ValueString())
		}
	}
}

// TestPipelineAutoDiscovery_EmptyMarshalsToList asserts an unset auto-discovery
// list produces a non-nil empty slice on the API payload (marshals to [], not
// null), matching the backend's default and avoiding null-vs-[] churn.
func TestPipelineAutoDiscovery_EmptyMarshalsToList(t *testing.T) {
	r := &PipelineResource{client: &fakeTransformClient{}}
	model := PipelineResourceModel{
		Name:              types.StringValue("p"),
		SnapshotNewTables: types.BoolValue(true),
		Source:            &PipelineSourceModel{ID: types.StringValue("s1"), Name: types.StringValue("s"), Connector: types.StringValue("dynamodb"), Topics: setOfStrings(t)},
		Destination:       &PipelineDestinationModel{ID: types.StringValue("d1"), Name: types.StringValue("d"), Connector: types.StringValue("postgresql")},
		Tags:              setOfStrings(t),
	}
	apiObj, err := r.model2API(context.Background(), model)
	if err != nil {
		t.Fatalf("model2API: %v", err)
	}
	if apiObj.TopicAutoDiscoveryTransforms == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(apiObj.TopicAutoDiscoveryTransforms) != 0 {
		t.Fatalf("expected empty slice, got %v", apiObj.TopicAutoDiscoveryTransforms)
	}
}

func TestPreservePeriodicAuditRoundTripsUnchanged(t *testing.T) {
	fixDeletes := true
	existing := &api.PipelinePeriodicAudit{
		Topics:          []string{"public.orders", "public.customers"},
		TimestampColumn: "updated_at",
		IntervalMinutes: 60,
		FixDeletesOnly:  &fixDeletes,
	}

	var diags diag.Diagnostics
	got := preservePeriodicAudit(existing, []string{"public.customers", "public.orders"}, &diags)
	if got == nil {
		t.Fatal("an audit whose topics are all still streamed must be preserved")
	}
	if len(diags) != 0 {
		t.Fatalf("unchanged audit must not warn, got %v", diags)
	}
	if len(got.Topics) != 2 {
		t.Fatalf("topics = %v, want both preserved", got.Topics)
	}
	if got.TimestampColumn != "updated_at" || got.IntervalMinutes != 60 {
		t.Fatalf("audit settings not carried through: %+v", got)
	}
	if got.FixDeletesOnly == nil || !*got.FixDeletesOnly {
		t.Fatal("fix_deletes_only must round-trip")
	}
	if existing.Topics[0] != "public.orders" {
		t.Fatal("the caller's audit must not be mutated")
	}
}

func TestPreservePeriodicAuditDropsUnstreamedTopics(t *testing.T) {
	existing := &api.PipelinePeriodicAudit{
		Topics:          []string{"public.orders", "public.removed", "public.gone"},
		TimestampColumn: "updated_at",
		IntervalMinutes: 30,
	}

	var diags diag.Diagnostics
	got := preservePeriodicAudit(existing, []string{"public.orders"}, &diags)
	if got == nil {
		t.Fatal("an audit with at least one streamed topic must survive")
	}
	if len(got.Topics) != 1 || got.Topics[0] != "public.orders" {
		t.Fatalf("topics = %v, want only the streamed one", got.Topics)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warning count = %d, want 1", diags.WarningsCount())
	}
	if detail := diags.Warnings()[0].Detail(); !strings.Contains(detail, "public.gone, public.removed") {
		t.Fatalf("warning must name dropped topics in sorted order, got %q", detail)
	}
}

func TestPreservePeriodicAuditRemovedWhenNoTopicsRemain(t *testing.T) {
	existing := &api.PipelinePeriodicAudit{
		Topics:          []string{"public.removed"},
		TimestampColumn: "updated_at",
		IntervalMinutes: 30,
	}

	var diags diag.Diagnostics
	if got := preservePeriodicAudit(existing, []string{"public.orders"}, &diags); got != nil {
		t.Fatalf("audit covering no streamed topic must be dropped, got %+v", got)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warning count = %d, want 1", diags.WarningsCount())
	}
}

func TestPreservePeriodicAuditNilIsNoop(t *testing.T) {
	var diags diag.Diagnostics
	if got := preservePeriodicAudit(nil, []string{"public.orders"}, &diags); got != nil {
		t.Fatal("no audit configured must stay nil")
	}
	if len(diags) != 0 {
		t.Fatalf("no audit must not warn, got %v", diags)
	}
}

// The backend answers 400 for a pipeline over an API source; the plan says so first and names the resource to use.
func TestAPISourceConnectorIsRefusedAtPlan(t *testing.T) {
	for _, code := range []string{"hubspot", "salesforce", "netsuite", "stripe", "zendesk", "google_analytics", "facebook_ads", "google_ads"} {
		problem := apiSourceProblem(code)
		if problem == "" || !strings.Contains(problem, "streamkap_topic_destination") {
			t.Errorf("%s: %q does not refuse the API source or name streamkap_topic_destination", code, problem)
		}
	}
	for _, code := range []string{"postgresql", "mysql", "stripe_webhook", ""} {
		if problem := apiSourceProblem(code); problem != "" {
			t.Errorf("%s is not an API source, refused with %q", code, problem)
		}
	}
}

// A managed pipeline is read-only (409 on update and delete), so the resource must not adopt one.
func TestManagedPipelineIsNotAdopted(t *testing.T) {
	if err := managedPipelineError(api.Pipeline{ID: "p1", Destination: api.PipelineDestination{ID: "d1"}}); err != nil {
		t.Fatalf("an unmanaged pipeline was refused: %v", err)
	}
	err := managedPipelineError(api.Pipeline{ID: "p1", Managed: true, Destination: api.PipelineDestination{ID: "d1"}})
	if err == nil {
		t.Fatal("a managed pipeline was adopted")
	}
	for _, want := range []string{"streamkap_topic_destination", "<topic_id>|d1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%q does not say %q", err, want)
		}
	}
}

type managedPipelineClient struct {
	api.StreamkapAPI
}

func (managedPipelineClient) GetPipeline(_ context.Context, id string) (*api.Pipeline, error) {
	return &api.Pipeline{ID: id, Managed: true, Destination: api.PipelineDestination{ID: "d1"}}, nil
}

// pipelineValue is the resource's object with every attribute null but the ones given.
func pipelineValue(t *testing.T, set map[string]tftypes.Value) (res.SchemaResponse, tftypes.Value) {
	t.Helper()
	var schemaResp res.SchemaResponse
	(&PipelineResource{}).Schema(context.Background(), res.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	for name, value := range set {
		values[name] = value
	}
	return schemaResp, tftypes.NewValue(objType, values)
}

// Refresh drops a managed pipeline from state with a warning instead of failing, which would block every plan and
// destroy in the workspace.
func TestReadDropsAManagedPipeline(t *testing.T) {
	schemaResp, raw := pipelineValue(t, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "p1")})
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: raw}
	resp := res.ReadResponse{State: state}
	(&PipelineResource{client: managedPipelineClient{}}).Read(context.Background(), res.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a managed pipeline failed the refresh: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("a managed pipeline was kept in state")
	}
	warnings := resp.Diagnostics.Warnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0].Detail(), "streamkap_topic_destination") {
		t.Fatalf("the removal must be explained with a warning naming streamkap_topic_destination: %v", resp.Diagnostics)
	}
}

func TestValidateConfigRefusesAnAPISource(t *testing.T) {
	var schemaResp res.SchemaResponse
	(&PipelineResource{}).Schema(context.Background(), res.SchemaRequest{}, &schemaResp)
	sourceType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object).AttributeTypes["source"].(tftypes.Object)
	source := func(connector string) tftypes.Value {
		return tftypes.NewValue(sourceType, map[string]tftypes.Value{
			"id":        tftypes.NewValue(tftypes.String, "s1"),
			"name":      tftypes.NewValue(tftypes.String, "src"),
			"connector": tftypes.NewValue(tftypes.String, connector),
			"topics":    tftypes.NewValue(sourceType.AttributeTypes["topics"], nil),
		})
	}
	validate := func(value tftypes.Value) diag.Diagnostics {
		_, raw := pipelineValue(t, map[string]tftypes.Value{"source": value})
		resp := res.ValidateConfigResponse{}
		(&PipelineResource{}).ValidateConfig(context.Background(), res.ValidateConfigRequest{
			Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
		}, &resp)
		return resp.Diagnostics
	}
	if diags := validate(source("hubspot")); !diags.HasError() {
		t.Error("a pipeline over an API source passed validation")
	}
	for name, value := range map[string]tftypes.Value{
		"a CDC source":         source("postgresql"),
		"an unknown source":    tftypes.NewValue(sourceType, tftypes.UnknownValue),
		"an unknown connector": tftypes.NewValue(sourceType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "s1"), "name": tftypes.NewValue(tftypes.String, "src"), "connector": tftypes.NewValue(tftypes.String, tftypes.UnknownValue), "topics": tftypes.NewValue(sourceType.AttributeTypes["topics"], nil)}),
		"no source at all yet": tftypes.NewValue(sourceType, nil),
	} {
		if diags := validate(value); diags.HasError() {
			t.Errorf("%s was refused: %v", name, diags)
		}
	}
}

type noCreateClient struct {
	api.StreamkapAPI
	t *testing.T
}

func (c noCreateClient) CreatePipeline(context.Context, api.Pipeline) (*api.Pipeline, error) {
	c.t.Fatal("a pipeline over an API source was sent to the backend")
	return nil, nil
}

// source.connector is unknown at plan time when it references a source created in the same apply, so ValidateConfig
// cannot see it; Create refuses it before the POST.
func TestCreateRefusesAnAPISourceBeforeThePOST(t *testing.T) {
	schemaResp, raw := pipelineValue(t, nil)
	sourceType := raw.Type().(tftypes.Object).AttributeTypes["source"].(tftypes.Object)
	destinationType := raw.Type().(tftypes.Object).AttributeTypes["destination"].(tftypes.Object)
	_, raw = pipelineValue(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "p"),
		"destination": tftypes.NewValue(destinationType, map[string]tftypes.Value{
			"id":        tftypes.NewValue(tftypes.String, "d1"),
			"name":      tftypes.NewValue(tftypes.String, "warehouse"),
			"connector": tftypes.NewValue(tftypes.String, "snowflake"),
		}),
		"source": tftypes.NewValue(sourceType, map[string]tftypes.Value{
			"id":        tftypes.NewValue(tftypes.String, "s1"),
			"name":      tftypes.NewValue(tftypes.String, "crm"),
			"connector": tftypes.NewValue(tftypes.String, "hubspot"),
			"topics":    tftypes.NewValue(sourceType.AttributeTypes["topics"], nil),
		}),
	})
	resp := res.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&PipelineResource{client: noCreateClient{t: t}}).Create(context.Background(), res.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw}, Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
	}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "streamkap_topic_destination") {
		t.Fatalf("an API source must be refused at apply: %v", resp.Diagnostics)
	}
}

func TestImportRefusesAManagedPipeline(t *testing.T) {
	schemaResp, _ := pipelineValue(t, nil)
	objType := schemaResp.Schema.Type().TerraformType(context.Background())
	resp := res.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	(&PipelineResource{client: managedPipelineClient{}}).ImportState(context.Background(), res.ImportStateRequest{ID: "p1"}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "streamkap_topic_destination") {
		t.Fatalf("importing a managed pipeline must fail with the reason: %v", resp.Diagnostics)
	}
}
