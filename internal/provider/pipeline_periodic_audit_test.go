package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// TestAccPipeline_PeriodicAuditSurvivesUpdate covers the backend trap recorded
// in AGENTS.md: periodic_audit is a conditional entity field, so a PUT that
// omits it deletes an audit configured outside Terraform. The audit is set
// through the API between steps (standing in for the UI), then the pipeline is
// updated three ways:
//
//  1. an unrelated change (rename) must leave the audit untouched;
//  2. dropping one audited topic must narrow the audit to the remaining topic;
//  3. dropping every audited topic must release the audit instead of failing
//     the apply on the backend's subset validation.
func TestAccPipeline_PeriodicAuditSurvivesUpdate(t *testing.T) {
	sourceName := acctestName(t, "audit-source")
	destinationName := acctestName(t, "audit-destination")
	pipelineName := acctestName(t, "audit-pipeline")

	connectorsDef := pipelineSrcPostgreSQLResourceDef(sourceName) +
		pipelineDestSnowflakeResourceDef(destinationName)

	pipelineDef := func(name string, topics ...string) string {
		quoted := make([]string, len(topics))
		for i, topic := range topics {
			quoted[i] = fmt.Sprintf("%q", topic)
		}
		return providerConfig + connectorsDef + fmt.Sprintf(`
resource "streamkap_pipeline" "test" {
	name                = %q
	snapshot_new_tables = false
	source = {
		id        = streamkap_source_postgresql.test.id
		name      = streamkap_source_postgresql.test.name
		connector = streamkap_source_postgresql.test.connector
		topics    = [%s]
	}
	destination = {
		id        = streamkap_destination_snowflake.test.id
		name      = streamkap_destination_snowflake.test.name
		connector = streamkap_destination_snowflake.test.connector
	}
}
`, name, strings.Join(quoted, ", "))
	}

	var pipelineID string
	capturePipelineID := func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["streamkap_pipeline.test"]
		if !ok {
			return fmt.Errorf("streamkap_pipeline.test not in state")
		}
		pipelineID = rs.Primary.ID
		return nil
	}

	// configureAudit stands in for the UI: it attaches a periodic audit covering
	// the given topics to the pipeline directly through the API.
	configureAudit := func(topics ...string) func() {
		return func() {
			client, err := testAccCheckDestroyClient()
			if err != nil {
				t.Fatalf("api client: %s", err)
			}
			ctx := context.Background()
			live, err := client.GetPipeline(ctx, pipelineID)
			if err != nil {
				t.Fatalf("get pipeline %s: %s", pipelineID, err)
			}
			// GET returns source topics prefixed with source_<id>.; the update
			// endpoint expects the pretty names, exactly as the provider sends them.
			for i, topic := range live.Source.Topics {
				live.Source.Topics[i] = strings.TrimPrefix(topic, "source_"+live.Source.ID+".")
			}
			live.PeriodicAudit = &api.PipelinePeriodicAudit{
				Topics:          topics,
				TimestampColumn: "updated_at",
				IntervalMinutes: 60,
			}
			if _, err := client.UpdatePipeline(ctx, pipelineID, *live); err != nil {
				t.Fatalf("configure periodic audit out of band: %s", err)
			}
		}
	}

	expectAuditTopics := func(want ...string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			client, err := testAccCheckDestroyClient()
			if err != nil {
				return err
			}
			live, err := client.GetPipeline(context.Background(), pipelineID)
			if err != nil {
				return fmt.Errorf("get pipeline %s: %w", pipelineID, err)
			}
			if len(want) == 0 {
				if live.PeriodicAudit != nil {
					return fmt.Errorf("periodic audit should have been released, still covers %v", live.PeriodicAudit.Topics)
				}
				return nil
			}
			if live.PeriodicAudit == nil {
				return fmt.Errorf("periodic audit was wiped by the Terraform update; wanted topics %v", want)
			}
			got := append([]string(nil), live.PeriodicAudit.Topics...)
			sort.Strings(got)
			sort.Strings(want)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				return fmt.Errorf("periodic audit topics = %v, want %v", got, want)
			}
			if live.PeriodicAudit.TimestampColumn != "updated_at" || live.PeriodicAudit.IntervalMinutes != 60 {
				return fmt.Errorf("periodic audit settings changed: %+v", live.PeriodicAudit)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: pipelineDef(pipelineName, "streamkap.customer", "streamkap.customer2"),
				Check:  capturePipelineID,
			},
			// Unrelated update: rename. The audit set out of band must survive.
			{
				PreConfig: configureAudit("streamkap.customer", "streamkap.customer2"),
				Config:    pipelineDef(pipelineName+"-renamed", "streamkap.customer", "streamkap.customer2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_pipeline.test", "name", pipelineName+"-renamed"),
					expectAuditTopics("streamkap.customer", "streamkap.customer2"),
				),
			},
			// Drop one audited topic: the audit narrows instead of failing.
			{
				Config: pipelineDef(pipelineName+"-renamed", "streamkap.customer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_pipeline.test", "source.topics.#", "1"),
					expectAuditTopics("streamkap.customer"),
				),
			},
			// Drop every audited topic: the audit is released, the apply succeeds.
			{
				Config: pipelineDef(pipelineName+"-renamed", "streamkap.customer2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_pipeline.test", "source.topics.#", "1"),
					expectAuditTopics(),
				),
			},
		},
	})
}
