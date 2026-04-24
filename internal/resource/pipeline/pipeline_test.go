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
