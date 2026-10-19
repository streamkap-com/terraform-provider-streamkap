package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

type predicateGraphClient struct {
	responses []*api.SMTChainRead
	writes    []api.SMTChainWrite
}

func (c *predicateGraphClient) GetSMTChain(context.Context, string, string) (*api.SMTChainRead, error) {
	return nil, fmt.Errorf("unexpected GetSMTChain")
}

func (c *predicateGraphClient) GetSMTChainAdoption(context.Context, string, string) (*api.SMTChainAdoptionRead, error) {
	return nil, fmt.Errorf("unexpected GetSMTChainAdoption")
}

func (c *predicateGraphClient) PutSMTChain(_ context.Context, _ string, _ string, write api.SMTChainWrite) (*api.SMTChainRead, error) {
	c.writes = append(c.writes, write)
	if len(c.responses) == 0 {
		return nil, fmt.Errorf("unexpected PutSMTChain")
	}
	response := c.responses[0]
	c.responses = c.responses[1:]
	return response, nil
}

func predicateFixtureValue(t *testing.T, chain *ChainSchema, key, typeID, pattern, id string, version int64, configured []string) types.Object {
	t.Helper()
	childType := chain.PredicateEntryType.AttributeTypes()[typeID].(types.ObjectType)
	child := objectWith(t, childType, map[string]attr.Value{"pattern": types.StringValue(pattern)})
	values := map[string]attr.Value{attrKey: types.StringValue(key), attrType: types.StringValue(typeID), typeID: child}
	if id != "" {
		values[attrID] = types.StringValue(id)
		values[attrAlias] = types.StringValue("Predicate1")
		values[attrSchemaVersion] = types.Int64Value(1)
		values[attrCodecVersion] = types.Int64Value(1)
	}
	values[attrSecretVersion] = types.Int64Value(version)
	if configured != nil {
		values[attrConfiguredSecretPaths] = types.ListValueMust(types.StringType, stringValues(configured))
	}
	return objectWith(t, chain.PredicateEntryType, values)
}

func transformFixtureValue(t *testing.T, chain *ChainSchema, key, id, replacement, predicateKey string, negate bool) types.Object {
	t.Helper()
	childType := chain.EntryType.AttributeTypes()["route"].(types.ObjectType)
	child := objectWith(t, childType, map[string]attr.Value{"replacement": types.StringValue(replacement)})
	values := map[string]attr.Value{attrKey: types.StringValue(key), attrType: types.StringValue("route"), attrName: types.StringValue(key), attrEnabled: types.BoolValue(true), "route": child}
	if id != "" {
		values[attrID] = types.StringValue(id)
		values[attrAlias] = types.StringValue("Route" + key)
		values[attrSchemaVersion] = types.Int64Value(1)
		values[attrCodecVersion] = types.Int64Value(1)
	}
	values[attrSecretVersion] = types.Int64Value(0)
	if predicateKey != "" {
		attachmentType := chain.EntryType.AttributeTypes()[attrPredicate].(types.ObjectType)
		values[attrPredicate] = objectWith(t, attachmentType, map[string]attr.Value{attrKey: types.StringValue(predicateKey), attrNegate: types.BoolValue(negate)})
	}
	return objectWith(t, chain.EntryType, values)
}

func stringValues(values []string) []attr.Value {
	out := make([]attr.Value, len(values))
	for i, value := range values {
		out[i] = types.StringValue(value)
	}
	return out
}

func predicateSecretFixture(t *testing.T, key, pointer, value string, version int64) types.List {
	t.Helper()
	entryType := PredicateSecretsAttribute().NestedObject.Type().(types.ObjectType)
	valueType := entryType.AttributeTypes()[attrValues].(types.ListType).ElemType.(types.ObjectType)
	operation := objectWith(t, valueType, map[string]attr.Value{
		attrPointer: types.StringValue(pointer),
		attrValue:   types.StringValue(value),
		attrClear:   types.BoolValue(false),
	})
	entry := objectWith(t, entryType, map[string]attr.Value{
		attrKey:     types.StringValue(key),
		attrVersion: types.Int64Value(version),
		attrValues:  types.ListValueMust(valueType, []attr.Value{operation}),
	})
	return types.ListValueMust(entryType, []attr.Value{entry})
}

func TestPredicateStateKeepsOneSharedRecordAndBothAttachments(t *testing.T) {
	s := predicateSurfaceFixture(t)
	var read api.SMTChainRead
	require.NoError(t, json.Unmarshal([]byte(`{"desired":{"chain_revision":"r2","deployment_generation":2,"predicates":[{"id":"predicate-p","alias":"Predicate1","type":"topic_match_secret","schema_version":1,"codec_version":1,"config":{"pattern":"^orders\\..*$"},"configured_secret_paths":["/token"]}],"transforms":[{"kind":"typed","id":"a","alias":"Routea","type":"route","schema_version":1,"codec_version":1,"name":"a","enabled":true,"config":{"replacement":"a"},"configured_secret_paths":[],"predicate":{"id":"predicate-p","negate":false},"selector":null},{"kind":"typed","id":"b","alias":"Routeb","type":"route","schema_version":1,"codec_version":1,"name":"b","enabled":true,"config":{"replacement":"b"},"configured_secret_paths":[],"predicate":{"id":"predicate-p","negate":true},"selector":null}],"managed":[],"managed_overrides":{},"order":[]},"last_applied":null,"application_status":"pending"}`), &read))

	state, diags := s.stateGraphFrom(context.Background(), &read, []string{"a", "b"}, []string{"p"}, map[string]int64{"a": 0, "b": 0}, map[string]int64{"p": 7})
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, state.Predicates.Elements(), 1)
	predicate := state.Predicates.Elements()[0].(types.Object).Attributes()
	require.Equal(t, "p", str(predicate[attrKey]))
	require.Equal(t, "predicate-p", str(predicate[attrID]))
	require.Equal(t, int64(7), i64(predicate[attrSecretVersion]))
	require.Equal(t, []attr.Value{types.StringValue("/token")}, predicate[attrConfiguredSecretPaths].(types.List).Elements())
	require.NotContains(t, predicate["topic_match_secret"].(types.Object).Attributes(), "token")
	for i, expectedNegate := range []bool{false, true} {
		attachment := state.Transforms.Elements()[i].(types.Object).Attributes()[attrPredicate].(types.Object).Attributes()
		require.Equal(t, "p", str(attachment[attrKey]))
		require.Equal(t, expectedNegate, boolValue(attachment[attrNegate]))
	}
}

func TestPredicateProjectionPinsSharedCreateCopyOnWriteNegateAndGC(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain

	newP := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicateFixtureValue(t, chain, "p", "topic_match", "orders", "", 0, nil)})
	newTransforms := types.ListValueMust(chain.EntryType, []attr.Value{
		transformFixtureValue(t, chain, "a", "a", "a", "p", false),
		transformFixtureValue(t, chain, "b", "b", "b", "p", false),
	})
	predicateCorrelation, d := correlateAt(AttrPredicates, nil, PlannedFromList(newP))
	require.False(t, d.HasError())
	predicateWrites, d := chain.ProjectPredicates(newP, predicateCorrelation, nil)
	require.False(t, d.HasError())
	transformWrites, d := chain.ProjectGraph(newTransforms, []Correlated{{Key: "a", Type: "route", ID: "a", Matched: true}, {Key: "b", Type: "route", ID: "b", Matched: true}}, nil, map[string]Correlated{"p": predicateCorrelation[0]})
	require.False(t, d.HasError())
	require.Equal(t, []api.SMTPredicateWrite{{RequestKey: "p", Type: "topic_match", Config: map[string]any{"pattern": "orders"}}}, predicateWrites)
	require.Equal(t, "p", transformWrites[0].Predicate.RequestKey)
	require.Equal(t, "p", transformWrites[1].Predicate.RequestKey)

	priorP := predicateFixtureValue(t, chain, "p", "topic_match", "orders", "predicate-p", 4, []string{"/token"})
	q := predicateFixtureValue(t, chain, "q", "topic_match", "payments", "", 0, nil)
	cow := types.ListValueMust(chain.PredicateEntryType, []attr.Value{priorP, q})
	cowCorrelated, d := correlateAt(AttrPredicates, PriorFromList(types.ListValueMust(chain.PredicateEntryType, []attr.Value{priorP})), PlannedFromList(cow))
	require.False(t, d.HasError())
	cowWrites, d := chain.ProjectPredicates(cow, cowCorrelated, nil)
	require.False(t, d.HasError())
	require.Equal(t, api.SMTPredicateWrite{ID: "predicate-p"}, cowWrites[0])
	require.Equal(t, api.SMTPredicateWrite{RequestKey: "q", Type: "topic_match", Config: map[string]any{"pattern": "payments"}}, cowWrites[1])
	require.Empty(t, cowWrites[1].SecretOperations, "Q cannot inherit P's configured secret presence")

	empty, d := chain.ProjectPredicates(types.ListValueMust(chain.PredicateEntryType, nil), nil, nil)
	require.False(t, d.HasError())
	require.NotNil(t, empty)
	require.Empty(t, empty, "removing the last reference sends an explicit empty predicate pool")
}

func TestPredicatePlanWriteRetainsImmutableRecordAndChangesOnlyNegate(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	priorPredicate := predicateFixtureValue(t, chain, "p", "topic_match", "orders", "predicate-p", 4, nil)
	priorPredicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{priorPredicate})
	priorTransform := transformFixtureValue(t, chain, "a", "transform-a", "route", "p", false)
	plannedTransform := transformFixtureValue(t, chain, "a", "transform-a", "route", "p", true)
	priorTransforms := types.ListValueMust(chain.EntryType, []attr.Value{priorTransform})
	plannedTransforms := types.ListValueMust(chain.EntryType, []attr.Value{plannedTransform})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{
		AttrChain:      plannedTransforms,
		AttrPredicates: priorPredicates,
		AttrRevision:   types.StringValue("r1"),
	})
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

	write, diags := s.PlanWrite(context.Background(), plan, priorTransforms, cfg, priorPredicates, types.MapNull(types.BoolType))
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, []api.SMTPredicateWrite{{ID: "predicate-p"}}, write.Predicates)
	require.Equal(t, &api.SMTPredicateAttachment{ID: "predicate-p", Negate: true}, write.Transforms[0].Predicate)
	require.Empty(t, write.Predicates[0].Config)
	require.Empty(t, write.Predicates[0].SecretOperations)
}

func TestPredicatePlanWriteCopiesOneAttachmentAndGarbageCollectsLastReference(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	p := predicateFixtureValue(t, chain, "p", "topic_match_secret", "orders", "predicate-p", 3, []string{"/token"})
	priorPredicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{p})
	priorTransforms := types.ListValueMust(chain.EntryType, []attr.Value{
		transformFixtureValue(t, chain, "a", "transform-a", "a", "p", false),
		transformFixtureValue(t, chain, "b", "transform-b", "b", "p", false),
	})
	q := predicateFixtureValue(t, chain, "q", "topic_match_secret", "payments", "", 0, nil)
	plannedPredicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{p, q})
	plannedTransforms := types.ListValueMust(chain.EntryType, []attr.Value{
		transformFixtureValue(t, chain, "a", "transform-a", "a", "q", false),
		transformFixtureValue(t, chain, "b", "transform-b", "b", "p", false),
	})
	_, resourceSchema := testSurface(chain)
	qSecrets := predicateSecretFixture(t, "q", "/token", "new-q-secret", 1)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: plannedTransforms, AttrPredicates: plannedPredicates, AttrPredicateSecrets: qSecrets, AttrRevision: types.StringValue("r1")})
	write, diags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, priorTransforms, cfg, priorPredicates, types.MapNull(types.BoolType))
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, []api.SMTPredicateWrite{{ID: "predicate-p"}, {RequestKey: "q", Type: "topic_match_secret", Config: map[string]any{"pattern": "payments"}, SecretOperations: []api.SMTSecretOperation{{Pointer: "/token", Operation: api.SMTSecretReplace, Value: "new-q-secret"}}}}, write.Predicates)
	require.Equal(t, &api.SMTPredicateAttachment{RequestKey: "q", Negate: false}, write.Transforms[0].Predicate)
	require.Equal(t, &api.SMTPredicateAttachment{ID: "predicate-p", Negate: false}, write.Transforms[1].Predicate)

	emptyPredicates := types.ListValueMust(chain.PredicateEntryType, nil)
	emptyTransforms := types.ListValueMust(chain.EntryType, nil)
	removeCfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: emptyTransforms, AttrPredicates: emptyPredicates, AttrRevision: types.StringValue("r2")})
	remove, removeDiags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: removeCfg.Raw}, plannedTransforms, removeCfg, plannedPredicates, types.MapNull(types.BoolType))
	require.False(t, removeDiags.HasError(), "%v", removeDiags)
	require.Empty(t, remove.Transforms)
	require.NotNil(t, remove.Predicates)
	require.Empty(t, remove.Predicates)
}

func TestPredicatePlanWriteRejectsEditingRetainedConfig(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	priorPredicate := predicateFixtureValue(t, chain, "p", "topic_match", "orders", "predicate-p", 4, nil)
	editedPredicate := predicateFixtureValue(t, chain, "p", "topic_match", "payments", "predicate-p", 4, nil)
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{editedPredicate})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "transform-a", "route", "p", false)})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrRevision: types.StringValue("r1")})
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

	_, diags := s.PlanWrite(context.Background(), plan, transforms, cfg, types.ListValueMust(chain.PredicateEntryType, []attr.Value{priorPredicate}), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Predicate records are immutable", diags.Errors()[0].Summary())
}

func TestFreshPredicateSecretIsOwnedByItsNewIdentity(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicate := predicateFixtureValue(t, chain, "q", "topic_match_secret", "payments", "", 0, nil)
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicate})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "transform-a", "route", "q", false)})
	secrets := predicateSecretFixture(t, "q", "/token", "new-q-secret", 1)
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrPredicateSecrets: secrets, AttrRevision: types.StringNull()})
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

	write, diags := s.PlanWrite(context.Background(), plan, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, []api.SMTSecretOperation{{Pointer: "/token", Operation: api.SMTSecretReplace, Value: "new-q-secret"}}, write.Predicates[0].SecretOperations)
	require.Equal(t, "q", write.Transforms[0].Predicate.RequestKey)
}

func TestPredicateImportKeysStayStableByServerID(t *testing.T) {
	reads := []api.SMTPredicateRead{{ID: "predicate-p"}, {ID: "predicate-q"}}
	require.Equal(t, []string{"predicate-p", "predicate-q"}, keysForPredicateReads(nil, reads))
	prior := []PriorInstance{{Key: "p", ID: "predicate-p"}, {Key: "q", ID: "predicate-q"}}
	require.Equal(t, []string{"q", "p"}, keysForPredicateReads(prior, []api.SMTPredicateRead{{ID: "predicate-q"}, {ID: "predicate-p"}}))
}

func TestTransformAndPredicateKeysOccupySeparateNamespaces(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicateFixtureValue(t, chain, "shared", "topic_match", "orders", "", 0, nil)})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "shared", "", "route", "shared", false)})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrRevision: types.StringNull()})

	diags := s.ValidateConfig(context.Background(), cfg)
	require.False(t, diags.HasError(), "%v", diags)
}

func TestPredicateRequiredSecretRefusesBeforeProjection(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicate := predicateFixtureValue(t, chain, "q", "topic_match_secret", "payments", "", 0, nil)
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicate})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "a", "a", "q", false)})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrRevision: types.StringNull()})
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}
	_, diags := s.PlanWrite(context.Background(), plan, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "New predicate needs its own secret", diags.Errors()[0].Summary())
}

func TestRetainedPredicateSecretRotationRequiresCopyOnWrite(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicate := predicateFixtureValue(t, chain, "p", "topic_match_secret", "orders", "predicate-p", 1, []string{"/token"})
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicate})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "transform-a", "route", "p", false)})
	secrets := predicateSecretFixture(t, "p", "/token", "edited-secret", 2)
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrPredicateSecrets: secrets, AttrRevision: types.StringValue("r1")})
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

	_, diags := s.PlanWrite(context.Background(), plan, transforms, cfg, predicates, types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Predicate records are immutable", diags.Errors()[0].Summary())
}

func TestManagedChainRequiresExplicitPredicatePool(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "", "route", "", false)})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: types.ListNull(chain.PredicateEntryType), AttrRevision: types.StringNull()})

	validateDiags := s.ValidateConfig(context.Background(), cfg)
	require.False(t, validateDiags.HasError(), "schema defaults are applied after config validation")
	_, planDiags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, planDiags.HasError())
	require.Equal(t, "Predicate graph must be explicit", planDiags.Errors()[0].Summary())

	explicit := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: types.ListValueMust(chain.PredicateEntryType, nil), AttrRevision: types.StringNull()})
	write, explicitDiags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: explicit.Raw}, types.ListNull(chain.EntryType), explicit, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.False(t, explicitDiags.HasError(), "%v", explicitDiags)
	require.NotNil(t, write.Predicates)
	require.Empty(t, write.Predicates)
}

func TestPredicateAttachmentCompatibilityAndNegateAreValidatedBeforeWrite(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*TypeSpec)
		want   string
	}{
		"incompatible transform": {mutate: func(spec *TypeSpec) { spec.CompatibleTransformTypes = nil }, want: "Predicate is incompatible with transform"},
		"unsupported negate":     {mutate: func(spec *TypeSpec) { spec.SupportsNegate = false }, want: "Predicate does not support negate"},
	} {
		t.Run(name, func(t *testing.T) {
			s := predicateSurfaceFixture(t)
			chain := s.Chain
			tc.mutate(currentSpec(t, chain.PredicateCatalog, "predicate", "topic_match"))
			predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicateFixtureValue(t, chain, "p", "topic_match", "orders", "", 0, nil)})
			transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "", "route", "p", true)})
			_, resourceSchema := testSurface(chain)
			cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrRevision: types.StringNull()})
			_, diags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
			require.True(t, diags.HasError())
			require.Equal(t, tc.want, diags.Errors()[0].Summary())
		})
	}
}

func TestPredicateUnknownGraphAndAttachmentRefuseBeforeHTTP(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	_, resourceSchema := testSurface(chain)
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "", "route", "", false)})
	unknownPool := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: types.ListUnknown(chain.PredicateEntryType), AttrRevision: types.StringNull()})
	_, poolDiags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: unknownPool.Raw}, types.ListNull(chain.EntryType), unknownPool, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, poolDiags.HasError())
	require.Equal(t, "Predicate graph is unknown", poolDiags.Errors()[0].Summary())

	predicate := predicateFixtureValue(t, chain, "p", "topic_match", "orders", "", 0, nil)
	unknownAttachment := transformFixtureValue(t, chain, "a", "", "route", "", false).Attributes()
	attachmentType := chain.EntryType.AttributeTypes()[attrPredicate].(types.ObjectType)
	unknownAttachment[attrPredicate] = types.ObjectUnknown(attachmentType.AttributeTypes())
	unknownTransforms := types.ListValueMust(chain.EntryType, []attr.Value{types.ObjectValueMust(chain.EntryType.AttributeTypes(), unknownAttachment)})
	unknownCfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: unknownTransforms, AttrPredicates: types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicate}), AttrRevision: types.StringNull()})
	_, attachmentDiags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: unknownCfg.Raw}, types.ListNull(chain.EntryType), unknownCfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, attachmentDiags.HasError())
	require.Equal(t, "Predicate attachment is unknown", attachmentDiags.Errors()[0].Summary())

	for name, tc := range map[string]struct {
		attachment types.Object
		want       string
	}{
		"key": {
			attachment: types.ObjectValueMust(attachmentType.AttributeTypes(), map[string]attr.Value{attrKey: types.StringUnknown(), attrNegate: types.BoolValue(false)}),
			want:       "Predicate attachment key is unknown",
		},
		"negate": {
			attachment: types.ObjectValueMust(attachmentType.AttributeTypes(), map[string]attr.Value{attrKey: types.StringValue("p"), attrNegate: types.BoolUnknown()}),
			want:       "Predicate attachment negate is unknown",
		},
	} {
		t.Run(name, func(t *testing.T) {
			entry := transformFixtureValue(t, chain, "a", "", "route", "", false).Attributes()
			entry[attrPredicate] = tc.attachment
			planned := types.ListValueMust(chain.EntryType, []attr.Value{types.ObjectValueMust(chain.EntryType.AttributeTypes(), entry)})
			cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: planned, AttrPredicates: types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicate}), AttrRevision: types.StringNull()})
			_, diags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
			require.True(t, diags.HasError())
			require.Equal(t, tc.want, diags.Errors()[0].Summary())
		})
	}
}

func TestUnreferencedPredicateRefusesBeforeHTTP(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicateFixtureValue(t, chain, "p", "topic_match", "orders", "", 0, nil)})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "", "route", "", false)})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrRevision: types.StringNull()})

	_, diags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Predicate has no attachment", diags.Errors()[0].Summary())
}

func TestPredicatePoolWithoutManagedChainRefusesBeforeHTTP(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicateFixtureValue(t, chain, "p", "topic_match", "orders", "", 0, nil)})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: types.ListNull(chain.EntryType), AttrPredicates: predicates, AttrRevision: types.StringNull()})

	_, diags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Predicate graph needs a managed chain", diags.Errors()[0].Summary())
}

func TestUnknownPredicatePoolWithoutManagedChainRefusesBeforeHTTP(t *testing.T) {
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: types.ListNull(chain.EntryType), AttrPredicates: types.ListUnknown(chain.PredicateEntryType), AttrRevision: types.StringNull()})

	_, diags := s.PlanWrite(context.Background(), tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Predicate graph is unknown", diags.Errors()[0].Summary())
}

func TestApplyGraphRejectsOnePredicateKeyWithTwoReturnedIDs(t *testing.T) {
	s := predicateSurfaceFixture(t)
	attachment := func(id string) json.RawMessage {
		encoded, err := json.Marshal(api.SMTPredicateAttachment{ID: id})
		require.NoError(t, err)
		return encoded
	}
	client := &predicateGraphClient{responses: []*api.SMTChainRead{
		{Desired: api.SMTChainSnapshot{
			Predicates: []api.SMTPredicateRead{{ID: "predicate-one", Type: "topic_match", Config: map[string]any{"pattern": "orders"}}, {ID: "predicate-two", Type: "topic_match", Config: map[string]any{"pattern": "orders"}}},
			Transforms: []api.SMTInstanceRead{{ID: "a", Predicate: attachment("predicate-one")}, {ID: "b", Predicate: attachment("predicate-two")}},
		}},
	}}
	write := &Write{
		Configured:          true,
		ChainConfigured:     true,
		Predicates:          []api.SMTPredicateWrite{{RequestKey: "p", Type: "topic_match", Config: map[string]any{"pattern": "orders"}}},
		Transforms:          []api.SMTInstanceWrite{{Predicate: &api.SMTPredicateAttachment{RequestKey: "p"}}, {Predicate: &api.SMTPredicateAttachment{RequestKey: "p"}}},
		correlated:          []Correlated{{Key: "a"}, {Key: "b"}},
		predicateCorrelated: []Correlated{{Key: "p"}},
		versions:            map[string]int64{},
		predicateVersions:   map[string]int64{},
	}

	_, diags := s.ApplyGraph(context.Background(), client, "destinations", "connector", write, types.StringNull())
	require.True(t, diags.HasError())
	require.Equal(t, "Predicate identity cannot be represented", diags.Errors()[0].Summary())
}

func TestApplyGraphPreservesSharedPredicateIdentityThroughCopyAndCleanup(t *testing.T) {
	s := predicateSurfaceFixture(t)
	predicateRead := func(id, pattern string, configured []string) api.SMTPredicateRead {
		return api.SMTPredicateRead{ID: id, Alias: "Predicate" + id, Type: "topic_match_secret", SchemaVersion: 1, CodecVersion: 1, Config: map[string]any{"pattern": pattern}, ConfiguredSecretPaths: configured}
	}
	transformRead := func(id, predicateID string) api.SMTInstanceRead {
		attachment, err := json.Marshal(api.SMTPredicateAttachment{ID: predicateID})
		require.NoError(t, err)
		return api.SMTInstanceRead{Kind: "typed", ID: id, Alias: "Route" + id, Type: "route", SchemaVersion: 1, CodecVersion: 1, Name: id, Enabled: true, Config: map[string]any{"replacement": id}, ConfiguredSecretPaths: []string{}, Predicate: attachment, Selector: json.RawMessage("null")}
	}
	read := func(revision string, predicates []api.SMTPredicateRead, transforms []api.SMTInstanceRead) *api.SMTChainRead {
		return &api.SMTChainRead{Desired: api.SMTChainSnapshot{ChainRevision: revision, Predicates: predicates, Transforms: transforms}, ApplicationStatus: "pending"}
	}
	client := &predicateGraphClient{responses: []*api.SMTChainRead{
		read("r1", []api.SMTPredicateRead{predicateRead("predicate-p", "orders", []string{"/token"})}, []api.SMTInstanceRead{transformRead("a", "predicate-p"), transformRead("b", "predicate-p")}),
		read("r2", []api.SMTPredicateRead{predicateRead("predicate-q", "payments", []string{}), predicateRead("predicate-p", "orders", []string{"/token"})}, []api.SMTInstanceRead{transformRead("a", "predicate-q"), transformRead("b", "predicate-p")}),
		read("r3", []api.SMTPredicateRead{predicateRead("predicate-p", "orders", []string{"/token"})}, []api.SMTInstanceRead{transformRead("b", "predicate-p")}),
		read("r4", []api.SMTPredicateRead{}, []api.SMTInstanceRead{}),
	}}
	writes := []*Write{
		{Configured: true, ChainConfigured: true, Predicates: []api.SMTPredicateWrite{{RequestKey: "p", Type: "topic_match_secret", Config: map[string]any{"pattern": "orders"}, SecretOperations: []api.SMTSecretOperation{{Pointer: "/token", Operation: api.SMTSecretReplace, Value: "p-secret"}}}}, Transforms: []api.SMTInstanceWrite{{Type: "route", Name: "a", Enabled: true, Config: map[string]any{"replacement": "a"}, Predicate: &api.SMTPredicateAttachment{RequestKey: "p"}}, {Type: "route", Name: "b", Enabled: true, Config: map[string]any{"replacement": "b"}, Predicate: &api.SMTPredicateAttachment{RequestKey: "p"}}}, correlated: []Correlated{{Key: "a"}, {Key: "b"}}, predicateCorrelated: []Correlated{{Key: "p"}}, versions: map[string]int64{}, predicateVersions: map[string]int64{"p": 1}},
		{Configured: true, ChainConfigured: true, Predicates: []api.SMTPredicateWrite{{ID: "predicate-p"}, {RequestKey: "q", Type: "topic_match_secret", Config: map[string]any{"pattern": "payments"}, SecretOperations: []api.SMTSecretOperation{{Pointer: "/token", Operation: api.SMTSecretReplace, Value: "q-secret"}}}}, Transforms: []api.SMTInstanceWrite{{ID: "a", Type: "route", Name: "a", Enabled: true, Config: map[string]any{"replacement": "a"}, Predicate: &api.SMTPredicateAttachment{RequestKey: "q"}}, {ID: "b", Type: "route", Name: "b", Enabled: true, Config: map[string]any{"replacement": "b"}, Predicate: &api.SMTPredicateAttachment{ID: "predicate-p"}}}, correlated: []Correlated{{Key: "a"}, {Key: "b"}}, predicateCorrelated: []Correlated{{Key: "p", ID: "predicate-p", Matched: true}, {Key: "q"}}, versions: map[string]int64{}, predicateVersions: map[string]int64{"p": 1, "q": 0}},
		{Configured: true, ChainConfigured: true, Predicates: []api.SMTPredicateWrite{{ID: "predicate-p"}}, Transforms: []api.SMTInstanceWrite{{ID: "b", Type: "route", Name: "b", Enabled: true, Config: map[string]any{"replacement": "b"}, Predicate: &api.SMTPredicateAttachment{ID: "predicate-p"}}}, correlated: []Correlated{{Key: "b"}}, predicateCorrelated: []Correlated{{Key: "p", ID: "predicate-p", Matched: true}}, versions: map[string]int64{}, predicateVersions: map[string]int64{"p": 1}},
		{Configured: true, ChainConfigured: true, Predicates: []api.SMTPredicateWrite{}, Transforms: []api.SMTInstanceWrite{}, correlated: []Correlated{}, predicateCorrelated: []Correlated{}, versions: map[string]int64{}, predicateVersions: map[string]int64{}},
	}
	var states []GraphState
	for i, write := range writes {
		expectedRevision := types.StringNull()
		if i > 0 {
			expectedRevision = types.StringValue(fmt.Sprintf("r%d", i))
		}
		state, diags := s.ApplyGraph(context.Background(), client, "destinations", "connector", write, expectedRevision)
		require.False(t, diags.HasError(), "%v", diags)
		states = append(states, state)
	}
	require.Len(t, states[0].Predicates.Elements(), 1)
	require.Equal(t, "p", (*client.writes[0].Transforms)[0].Predicate.RequestKey)
	require.Equal(t, "p", (*client.writes[0].Transforms)[1].Predicate.RequestKey)
	for _, transform := range states[0].Transforms.Elements() {
		require.Equal(t, "p", str(transform.(types.Object).Attributes()[attrPredicate].(types.Object).Attributes()[attrKey]))
	}
	require.Equal(t, []string{"q", "p"}, []string{str(states[1].Predicates.Elements()[0].(types.Object).Attributes()[attrKey]), str(states[1].Predicates.Elements()[1].(types.Object).Attributes()[attrKey])}, "provider keys follow predicate ids when the server reorders its pool")
	require.Equal(t, api.SMTPredicateWrite{ID: "predicate-p"}, (*client.writes[1].Predicates)[0])
	require.Equal(t, "q", (*client.writes[1].Transforms)[0].Predicate.RequestKey)
	require.Equal(t, "predicate-p", (*client.writes[1].Transforms)[1].Predicate.ID)
	require.Empty(t, states[1].Predicates.Elements()[0].(types.Object).Attributes()[attrConfiguredSecretPaths].(types.List).Elements(), "Q does not inherit P's configured secret presence")
	require.Equal(t, []attr.Value{types.StringValue("/token")}, states[1].Predicates.Elements()[1].(types.Object).Attributes()[attrConfiguredSecretPaths].(types.List).Elements())
	require.Len(t, states[2].Predicates.Elements(), 1, "removing A and Q preserves P while B still references it")
	require.Empty(t, states[3].Predicates.Elements(), "removing B and P sends explicit cleanup")
	require.NotNil(t, *client.writes[3].Predicates)
	require.Empty(t, *client.writes[3].Predicates)
}
