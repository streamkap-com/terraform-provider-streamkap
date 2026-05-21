package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
