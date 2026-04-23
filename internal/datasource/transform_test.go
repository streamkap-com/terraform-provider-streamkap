package datasource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// TestModelFromAPIObject_MismatchedTopicsAndIDs is a regression test for a
// runtime panic: the API returns topic_ids and topics as two parallel slices,
// but they can legitimately differ in length (e.g., topics not yet resolved).
// Previously topicMap[i].Name = apiObject.Topics[i] panicked when
// len(Topics) < len(TopicIDs).
func TestModelFromAPIObject_MismatchedTopicsAndIDs(t *testing.T) {
	ds := &TransformDataSource{}

	t.Run("fewer topics than ids", func(t *testing.T) {
		apiObject := api.Transform{
			ID:       "t-1",
			Name:     "example",
			TopicIDs: []string{"id-1", "id-2", "id-3"},
			Topics:   []string{"topic-a"}, // only one name resolved
		}
		model := &TransformDataSourceModel{}

		require.NotPanics(t, func() {
			ds.modelFromAPIObject(apiObject, model)
		})

		require.Len(t, model.TopicMap, 3)
		assert.Equal(t, "id-1", model.TopicMap[0].ID.ValueString())
		assert.Equal(t, "topic-a", model.TopicMap[0].Name.ValueString())
		assert.Equal(t, "id-2", model.TopicMap[1].ID.ValueString())
		assert.True(t, model.TopicMap[1].Name.IsNull(), "unpaired id should yield null name")
		assert.True(t, model.TopicMap[2].Name.IsNull())
	})

	t.Run("empty topics with populated ids", func(t *testing.T) {
		apiObject := api.Transform{
			ID:       "t-2",
			Name:     "example",
			TopicIDs: []string{"id-1"},
			Topics:   nil,
		}
		model := &TransformDataSourceModel{}

		require.NotPanics(t, func() {
			ds.modelFromAPIObject(apiObject, model)
		})

		require.Len(t, model.TopicMap, 1)
		assert.Equal(t, "id-1", model.TopicMap[0].ID.ValueString())
		assert.True(t, model.TopicMap[0].Name.IsNull())
	})

	t.Run("matching lengths behave unchanged", func(t *testing.T) {
		apiObject := api.Transform{
			ID:       "t-3",
			Name:     "example",
			TopicIDs: []string{"id-1", "id-2"},
			Topics:   []string{"topic-a", "topic-b"},
		}
		model := &TransformDataSourceModel{}

		ds.modelFromAPIObject(apiObject, model)

		require.Len(t, model.TopicMap, 2)
		assert.Equal(t, "topic-a", model.TopicMap[0].Name.ValueString())
		assert.Equal(t, "topic-b", model.TopicMap[1].Name.ValueString())
	})
}
