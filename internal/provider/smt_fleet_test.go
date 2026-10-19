package provider

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

var chainAttributes = []string{smt.AttrChain, smt.AttrSecrets, smt.AttrPredicates, smt.AttrPredicateSecrets, smt.AttrManagedOverrides, smt.AttrRevision}

type chainResource struct {
	typeName string
	config   connector.ConnectorConfig
	schema   rschema.Schema
}

// connectorResources are the registered connector resources with their
// runtime schema.
func connectorResources(t *testing.T) []chainResource {
	t.Helper()
	ctx := context.Background()
	var out []chainResource
	for _, factory := range (&streamkapProvider{}).Resources(ctx) {
		base, ok := factory().(*connector.BaseConnectorResource)
		if !ok {
			continue
		}
		meta := &resource.MetadataResponse{}
		base.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "streamkap"}, meta)
		schemaResp := &resource.SchemaResponse{}
		base.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		require.False(t, schemaResp.Diagnostics.HasError(), "%s: %v", meta.TypeName, schemaResp.Diagnostics)
		out = append(out, chainResource{typeName: meta.TypeName, config: base.Config(), schema: schemaResp.Schema})
	}
	require.NotEmpty(t, out)
	return out
}

// The backend can manage a chain on every published source and destination
// except kafkadirect (composition_context.potentially_supported, which gates
// the chain routes), and the pinned catalog publishes types for both hosts.
// Exactly those resources carry the chain, each built from its own host.
func TestSMTChainFleetFollowsBackendCapability(t *testing.T) {
	counts := map[connector.ConnectorType]int{}
	for _, r := range connectorResources(t) {
		t.Run(r.typeName, func(t *testing.T) {
			supported := r.config.GetConnectorCode() != "kafkadirect"
			for _, name := range chainAttributes {
				_, has := r.schema.Attributes[name]
				require.Equal(t, supported, has, "%s present=%v, backend support=%v", name, has, supported)
			}
			if !supported {
				return
			}
			counts[r.config.GetConnectorType()]++
			host := string(r.config.GetConnectorType()) + "s"
			want := generated.SMTChain(host)
			got := r.schema.Attributes[smt.AttrChain].(rschema.ListNestedAttribute).NestedObject.Attributes
			for _, typeID := range want.Catalog.TypeIDs() {
				require.Contains(t, got, typeID, "%s chain must offer %s", host, typeID)
			}
			own := map[string]bool{}
			for _, typeID := range want.Catalog.TypeIDs() {
				own[typeID] = true
			}
			for _, other := range []string{"destinations", "sources"} {
				if other == host {
					continue
				}
				for _, typeID := range generated.SMTChain(other).Catalog.TypeIDs() {
					if !own[typeID] {
						require.NotContains(t, got, typeID, "a %s chain must not offer the %s-only type %s", host, other, typeID)
					}
				}
			}
		})
	}
	require.Equal(t, 23, counts[connector.ConnectorTypeDestination], "destinations with a chain")
	require.Equal(t, 24, counts[connector.ConnectorTypeSource], "sources with a chain")
}

// A state written by the released v3 provider has no chain attributes. On
// every resource that now carries the chain it must plan no change at all: a
// chain attribute that planned a value would make the framework mark every
// computed null attribute unknown, and the released resource would never
// converge. Core's proposed new state is built here the way core builds it:
// the configured value where set, the prior value for a computed attribute,
// null otherwise.
func TestSMTChainReleasedStatePlansNoChangeOnEveryResource(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemas.Diagnostics)

	for _, r := range connectorResources(t) {
		if _, chained := r.schema.Attributes[smt.AttrChain]; !chained {
			continue
		}
		t.Run(r.typeName, func(t *testing.T) {
			objType := schemas.ResourceSchemas[r.typeName].ValueType().(tftypes.Object)
			generatedSchema := r.config.GetSchema()
			mappings := r.config.GetFieldMappings()
			prior := map[string]tftypes.Value{}
			config := map[string]tftypes.Value{}
			proposed := map[string]tftypes.Value{}
			for name, typ := range objType.AttributeTypes {
				prior[name], config[name], proposed[name] = tftypes.NewValue(typ, nil), tftypes.NewValue(typ, nil), tftypes.NewValue(typ, nil)
				a, released := r.schema.Attributes[name]
				if !released || isChainAttribute(name) {
					continue
				}
				switch {
				case a.IsRequired():
					v := placeholder(t, name, typ)
					prior[name], config[name], proposed[name] = v, v, v
				case a.IsComputed():
					if v, ok := releasedDefault(generatedSchema, mappings, name); ok {
						wire, err := smt.ToWire(v, path.Root(name))
						require.NoError(t, err)
						prior[name] = tftypes.NewValue(typ, toTerraformValue(t, typ, wire))
						proposed[name] = prior[name]
					}
				}
			}
			prior["id"] = tftypes.NewValue(tftypes.String, "existing")
			proposed["id"] = prior["id"]

			priorValue := tftypes.NewValue(objType, prior)
			priorDV, err := tfprotov6.NewDynamicValue(objType, priorValue)
			require.NoError(t, err)
			configDV, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, config))
			require.NoError(t, err)
			proposedDV, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, proposed))
			require.NoError(t, err)
			plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: r.typeName, PriorState: &priorDV, Config: &configDV, ProposedNewState: &proposedDV,
			})
			require.NoError(t, err)
			for _, d := range plan.Diagnostics {
				t.Errorf("plan diagnostic: %s: %s", d.Summary, d.Detail)
			}
			if t.Failed() {
				return
			}
			planned, err := plan.PlannedState.Unmarshal(objType)
			require.NoError(t, err)
			var plannedAttrs map[string]tftypes.Value
			require.NoError(t, planned.As(&plannedAttrs))
			var moved []string
			for name := range objType.AttributeTypes {
				if !prior[name].Equal(plannedAttrs[name]) {
					moved = append(moved, fmt.Sprintf("%s: %s -> %s", name, prior[name], plannedAttrs[name]))
				}
			}
			sort.Strings(moved)
			require.Empty(t, moved, "a released state must plan no change")
		})
	}
}

func isChainAttribute(name string) bool {
	for _, a := range chainAttributes {
		if a == name {
			return true
		}
	}
	return false
}

// releasedDefault is the value a released state holds for a defaulted
// attribute. The generated schema is the authority: the base strips the
// Default of a canonical attribute that has a deprecated alias and plans it
// in ModifyPlan instead, and the alias holds its canonical attribute's
// default.
func releasedDefault(generatedSchema rschema.Schema, mappings map[string]string, name string) (attr.Value, bool) {
	if v, ok := smt.DefaultValue(generatedSchema.Attributes[name]); ok {
		return v, true
	}
	a, ok := generatedSchema.Attributes[name]
	if ok && a.GetDeprecationMessage() == "" {
		return nil, false
	}
	for canonical, apiField := range mappings {
		if canonical != name && apiField == mappings[name] {
			if v, ok := smt.DefaultValue(generatedSchema.Attributes[canonical]); ok {
				return v, true
			}
		}
	}
	return nil, false
}

func placeholder(t *testing.T, name string, typ tftypes.Type) tftypes.Value {
	t.Helper()
	switch {
	case typ.Is(tftypes.String):
		// Valid JSON, so a jsontypes.Normalized attribute accepts it too.
		return tftypes.NewValue(typ, "{}")
	case typ.Is(tftypes.Number):
		return tftypes.NewValue(typ, 1)
	case typ.Is(tftypes.Bool):
		return tftypes.NewValue(typ, false)
	}
	t.Fatalf("required attribute %s has type %s; add a placeholder", name, typ)
	return tftypes.Value{}
}

// toTerraformValue converts a wire value from smt.ToWire back into the raw
// form tftypes.NewValue accepts for typ.
func toTerraformValue(t *testing.T, typ tftypes.Type, wire any) any {
	t.Helper()
	switch {
	case wire == nil:
		return nil
	case typ.Is(tftypes.String), typ.Is(tftypes.Bool):
		return wire
	case typ.Is(tftypes.Number):
		switch n := wire.(type) {
		case int64:
			return n
		case float64:
			return n
		case int:
			return n
		}
	case typ.Is(tftypes.List{}), typ.Is(tftypes.Set{}):
		items, ok := wire.([]any)
		require.True(t, ok, "%T", wire)
		var elem tftypes.Type
		if l, ok := typ.(tftypes.List); ok {
			elem = l.ElementType
		} else {
			elem = typ.(tftypes.Set).ElementType
		}
		out := make([]tftypes.Value, 0, len(items))
		for _, item := range items {
			out = append(out, tftypes.NewValue(elem, toTerraformValue(t, elem, item)))
		}
		return out
	}
	t.Fatalf("no conversion for %T into %s", wire, typ)
	return nil
}
