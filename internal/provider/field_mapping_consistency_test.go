package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/transform"
)

// The reflection bridge between schema, model and fieldMappings fails silently in
// both directions: shared.ConfigMapToModel skips a mapping key that doesn't
// resolve to a tfsdk field, and shared.GetStringField/SetStringField no-op on a Go
// field name that doesn't exist. A tfgen rename (the acronym table turning
// KcClusterId into KCClusterID, say) therefore blanks fields at runtime while
// every existing test stays green — smoke_test.go covers 6 of ~50 connectors and
// only asserts the mappings are non-empty strings.
//
// These tests close that hole for every registered resource. They are pure unit
// tests: schema + model + mappings, no API, no TF_ACC.

// reflectiveField is a Go field name a base resource looks up by reflection,
// paired with the framework type it must carry. Verified against the
// getStringField/setStringField/getStringSliceField call sites in
// internal/resource/{connector,transform}/base.go.
type reflectiveField struct {
	name    string
	typ     reflect.Type
	comment string
}

var (
	stringType = reflect.TypeOf(types.String{})
	setType    = reflect.TypeOf(types.Set{})
)

// connectorReflectiveFields — see connector/base.go Create/Read/Update/Delete.
var connectorReflectiveFields = []reflectiveField{
	{"ID", stringType, "read in Read/Update/Delete, written in Create"},
	{"Name", stringType, "read in Create/Update, written from the API response"},
	{"Connector", stringType, "written from the API response"},
	{"ConnectorStatus", stringType, "written from the API response"},
	{"KcClusterId", stringType, "sent on Create/Update, written on Read"},
	{"Tags", setType, "read and written on every CRUD call"},
}

// transformReflectiveFields — see transform/base.go Create/Read/Update/Delete.
var transformReflectiveFields = []reflectiveField{
	{"ID", stringType, "read in Read/Update/Delete, written in Create"},
	{"Name", stringType, "read in Create/Update, written from the API response"},
	{"TransformType", stringType, "written from the API response"},
	{"ConnectorStatus", stringType, "written to JobStatusUnknown before deploy refines it"},
	{"Tags", setType, "read and written on every CRUD call"},
}

// connectorBaseAttrs are the connector schema attributes the base resource maps
// by hand (they are top-level API fields, not entries of the connector's flat
// config map), so they legitimately have no fieldMappings row.
var connectorBaseAttrs = map[string]bool{
	"id":               true,
	"name":             true,
	"connector":        true,
	"connector_status": true,
	"kc_cluster_id":    true,
	"tags":             true,
}

// transformBaseAttrs are the transform equivalents. implementation_json, deploy
// and replay_window are injected by BaseTransformResource.Schema and serviced by
// dedicated API calls rather than the config map.
var transformBaseAttrs = map[string]bool{
	"id":                  true,
	"name":                true,
	"transform_type":      true,
	"connector_status":    true,
	"tags":                true,
	"implementation_json": true,
	"deploy":              true,
	"replay_window":       true,
}

// mappedResource is what both base resources expose for cross-checking.
type mappedResource struct {
	typeName  string
	schema    rschema.Schema
	mappings  map[string]string
	model     any
	baseAttrs map[string]bool
	reflected []reflectiveField
}

// mappedResources returns every registered resource that routes through the
// reflection bridge. Pipeline, topic and tag implement CRUD directly and are
// skipped.
func mappedResources(t *testing.T) []mappedResource {
	t.Helper()

	ctx := context.Background()
	p := &streamkapProvider{}

	var out []mappedResource
	for _, factory := range p.Resources(ctx) {
		res := factory()

		metaResp := &resource.MetadataResponse{}
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "streamkap"}, metaResp)

		schemaResp := &resource.SchemaResponse{}
		res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		require.False(t, schemaResp.Diagnostics.HasError(), "%s: schema should not have errors", metaResp.TypeName)

		switch r := res.(type) {
		case *connector.BaseConnectorResource:
			cfg := r.Config()
			out = append(out, mappedResource{
				typeName:  metaResp.TypeName,
				schema:    schemaResp.Schema,
				mappings:  cfg.GetFieldMappings(),
				model:     cfg.NewModelInstance(),
				baseAttrs: connectorBaseAttrs,
				reflected: connectorReflectiveFields,
			})
		case *transform.BaseTransformResource:
			cfg := r.Config()
			out = append(out, mappedResource{
				typeName:  metaResp.TypeName,
				schema:    schemaResp.Schema,
				mappings:  cfg.GetFieldMappings(),
				model:     cfg.NewModelInstance(),
				baseAttrs: transformBaseAttrs,
				reflected: transformReflectiveFields,
			})
		}
	}

	require.NotEmpty(t, out, "no reflection-bridged resources found; provider registration changed?")
	return out
}

// TestFieldMappingKeysResolveInModel asserts every fieldMappings key names a real
// tfsdk field. A key that doesn't resolve is dropped by ConfigMapToModel, so the
// attribute never reaches the API and never comes back — silently.
func TestFieldMappingKeysResolveInModel(t *testing.T) {
	for _, r := range mappedResources(t) {
		t.Run(r.typeName, func(t *testing.T) {
			_, tfsdkToField := shared.BuildTfsdkFieldIndex(r.model)

			require.NotEmpty(t, r.mappings, "resource has no field mappings")

			for tfAttr, apiField := range r.mappings {
				assert.Containsf(t, tfsdkToField, tfAttr,
					"field mapping %q -> %q has no tfsdk field in %T; "+
						"the marshaling bridge skips it, so the attribute is a no-op at runtime",
					tfAttr, apiField, r.model)
				assert.NotEmptyf(t, apiField, "field mapping %q has an empty API field name", tfAttr)
			}
		})
	}
}

// TestSchemaAttributesHaveFieldMapping asserts the reverse direction: every
// schema attribute a user can set is carried to the API by a mapping. An
// attribute without one accepts config and discards it.
func TestSchemaAttributesHaveFieldMapping(t *testing.T) {
	for _, r := range mappedResources(t) {
		t.Run(r.typeName, func(t *testing.T) {
			for attrName, attribute := range r.schema.Attributes {
				if r.baseAttrs[attrName] {
					continue
				}
				assert.Containsf(t, r.mappings, attrName,
					"schema attribute %q has no fieldMappings row: users can set it but "+
						"nothing sends it to the API. Add the mapping, or add the attribute "+
						"to the base-handled allowlist if the base resource services it directly.",
					attrName)

				// Deprecated aliases must target the same API field as their
				// replacement, not be quietly dropped from the mappings.
				if attribute.GetDeprecationMessage() != "" {
					assert.NotEmptyf(t, r.mappings[attrName],
						"deprecated alias %q has no API target; a v2 config setting it would be silently ignored",
						attrName)
				}
			}
		})
	}
}

// TestModelHasReflectiveFields asserts the Go field names the base resources look
// up by reflection exist with the expected type. shared.GetStringField returns ""
// and SetStringField no-ops when the name is wrong, so a rename in the generator's
// acronym table would blank id/name/connector across every connector with no test
// failure anywhere.
func TestModelHasReflectiveFields(t *testing.T) {
	for _, r := range mappedResources(t) {
		t.Run(r.typeName, func(t *testing.T) {
			v := reflect.ValueOf(r.model).Elem()

			for _, f := range r.reflected {
				field := v.FieldByName(f.name)
				if !assert.Truef(t, field.IsValid(),
					"model %T has no field %q (%s); the base resource resolves it by "+
						"reflection, which no-ops silently when the name is wrong",
					r.model, f.name, f.comment) {
					continue
				}
				assert.Equalf(t, f.typ, field.Type(),
					"model %T field %q has type %s, want %s", r.model, f.name, field.Type(), f.typ)
			}
		})
	}
}
