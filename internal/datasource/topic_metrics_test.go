package datasource

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func TestFlattenTopicTableMetricsPreservesNullsAndSortsRows(t *testing.T) {
	partitionCount := int64(3)
	recordErrorTotal := int64(0)
	metrics := api.TopicTableMetricsResponse{
		"topic-b": {
			ID: "topic-b",
		},
		"topic-a": {
			ID: "topic-a",
			Kafka: api.TopicTableKafkaMetrics{
				PartitionCount: &partitionCount,
			},
			SnapshotStatus:   []map[string]any{{"status": "completed"}},
			RecordErrorTotal: &recordErrorTotal,
		},
	}
	entities := []api.TopicMetricsEntity{{
		ID:       "source-a",
		TopicIDs: []string{"topic-a", "topic-b"},
	}}

	var diagnostics diag.Diagnostics
	results := flattenTopicTableMetrics(metrics, entities, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if got := results[0].TopicID.ValueString(); got != "topic-a" {
		t.Fatalf("first topic ID = %q, want topic-a", got)
	}
	if got := results[0].EntityID.ValueString(); got != "source-a" {
		t.Fatalf("entity ID = %q, want source-a", got)
	}
	if got := results[0].PartitionCount.ValueInt64(); got != 3 {
		t.Fatalf("partition count = %d, want 3", got)
	}
	if results[0].RecordErrorTotal.IsNull() || results[0].RecordErrorTotal.ValueInt64() != 0 {
		t.Fatalf("record error total should preserve a legitimate zero")
	}
	if got := results[0].SnapshotStatusJSON.ValueString(); got != `[{"status":"completed"}]` {
		t.Fatalf("snapshot status JSON = %q", got)
	}
	if !results[1].PartitionCount.IsNull() || !results[1].RecordErrorTotal.IsNull() {
		t.Fatalf("missing backend metrics must remain null")
	}
	if got := results[1].SnapshotStatusJSON.ValueString(); got != `[]` {
		t.Fatalf("absent snapshot status JSON = %q, want an empty array", got)
	}
	if !results[0].MessagesIn.IsNull() || !results[0].AvgLatencyMs.IsNull() {
		t.Fatalf("unavailable legacy metrics must remain null")
	}
}

func TestFlattenTopicTableMetricsAmbiguousEntity(t *testing.T) {
	metrics := api.TopicTableMetricsResponse{"shared": {ID: "shared"}, "unique": {ID: "unique"}}
	entities := []api.TopicMetricsEntity{
		{ID: "source", TopicIDs: []string{"shared", "unique"}},
		{ID: "source", TopicIDs: []string{"unique"}},
		{ID: "destination", TopicIDs: []string{"shared"}},
	}
	var diagnostics diag.Diagnostics
	results := flattenTopicTableMetrics(metrics, entities, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if !results[0].EntityID.IsNull() {
		t.Fatal("a shared topic must not be assigned to an arbitrary entity")
	}
	if got := results[1].EntityID.ValueString(); got != "source" {
		t.Fatalf("unique topic entity = %q, want source", got)
	}
}

func TestFlattenTopicTableMetricsWarnsOnUnreturnedTopics(t *testing.T) {
	metrics := api.TopicTableMetricsResponse{"owned": {ID: "owned"}}
	entities := []api.TopicMetricsEntity{{
		ID:       "source-a",
		TopicIDs: []string{"owned", "not-owned", "missing"},
	}}

	var diagnostics diag.Diagnostics
	results := flattenTopicTableMetrics(metrics, entities, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diagnostics)
	}
	if len(results) != 1 {
		t.Fatalf("result count = %d, want 1", len(results))
	}
	if diagnostics.WarningsCount() != 1 {
		t.Fatalf("warning count = %d, want 1", diagnostics.WarningsCount())
	}
	if detail := diagnostics.Warnings()[0].Detail(); !strings.Contains(detail, "missing, not-owned") {
		t.Fatalf("warning must name the dropped topics in sorted order, got %q", detail)
	}
}
