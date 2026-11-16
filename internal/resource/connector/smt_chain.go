package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

const smtChainAttr = "smt_chain"

// smtSecretMask is what the API returns in place of a stored secret.
const smtSecretMask = "*****"

type smtItemModel struct {
	ID           types.String         `tfsdk:"id"`
	Type         types.String         `tfsdk:"type"`
	Name         types.String         `tfsdk:"name"`
	Enabled      types.Bool           `tfsdk:"enabled"`
	Config       jsontypes.Normalized `tfsdk:"config"`
	SecretConfig types.Map            `tfsdk:"secret_config"`
	Predicate    *smtPredicateModel   `tfsdk:"predicate"`
}

type smtPredicateModel struct {
	Type   types.String         `tfsdk:"type"`
	Config jsontypes.Normalized `tfsdk:"config"`
	Negate types.Bool           `tfsdk:"negate"`
}

// smtChainAttribute is the smt_chain schema shared by every Kafka Connect
// source and destination.
func smtChainAttribute(host ConnectorType) schema.ListNestedAttribute {
	supported := "`filter`"
	if host == ConnectorTypeDestination {
		supported = "`regex_router`, `mask_field`, `drop_fields`"
	}
	configAttr := func(example string) schema.StringAttribute {
		d := "JSON object of properties, e.g. `jsonencode(" + example + ")`. Values are strings, booleans or lists of strings. Put secrets in `secret_config`."
		return schema.StringAttribute{Optional: true, CustomType: jsontypes.NormalizedType{}, Description: d, MarkdownDescription: d}
	}
	d := "Single Message Transforms applied by this connector, in the order they run (at most 20). Supported types: " + supported + ". Removing the attribute removes the chain."
	return schema.ListNestedAttribute{
		Optional:            true,
		Description:         d,
		MarkdownDescription: d,
		Validators:          []validator.List{listvalidator.SizeAtMost(20)},
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Kafka Connect alias the server assigns, e.g. `MaskField1`. Kept across plans while the item's type and name stay the same.",
			},
			"type":    schema.StringAttribute{Required: true, Description: "Transform type, e.g. `mask_field`."},
			"name":    schema.StringAttribute{Required: true, Description: "Display name."},
			"enabled": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "A disabled item keeps its settings but does not run. Defaults to `true`."},
			"config":  configAttr(`{ "fields.include.list" = ["public.users.email"] }`),
			"secret_config": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "Secret properties, e.g. `mask.salt`, kept out of plan output. **Security:** stored in state; mark the source variable sensitive. Removing a key clears the stored secret.",
			},
			"predicate": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Run the transform only on records that match.",
				Attributes: map[string]schema.Attribute{
					"type":   schema.StringAttribute{Required: true, Description: "Predicate type, e.g. `topic_name_matches`."},
					"config": configAttr(`{ pattern = "orders.*" }`),
					"negate": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Apply the transform to the records that do not match. Defaults to `false`."},
				},
			},
		}},
	}
}

// hasSmtChain reports whether the connector's model carries smt_chain (tfgen
// emits it for Kafka Connect connectors only).
func (r *BaseConnectorResource) hasSmtChain() bool {
	_, ok := reflect.TypeOf(r.config.NewModelInstance()).Elem().FieldByName("SmtChain")
	return ok
}

type attrGetter func(context.Context, path.Path, any) diag.Diagnostics

// getSmtChain returns nil when smt_chain is null or unknown.
func getSmtChain(ctx context.Context, get attrGetter) (*[]smtItemModel, diag.Diagnostics) {
	var list types.List
	diags := get(ctx, path.Root(smtChainAttr), &list)
	if diags.HasError() || list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	items := []smtItemModel{}
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	return &items, diags
}

// planSmtChainIDs keeps each item's id from prior state, matched by type and
// name, so reordering or editing other items does not change it.
func planSmtChainIDs(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() {
		return
	}
	planned, diags := getSmtChain(ctx, resp.Plan.GetAttribute)
	resp.Diagnostics.Append(diags...)
	prior, diags := getSmtChain(ctx, req.State.GetAttribute)
	resp.Diagnostics.Append(diags...)
	if planned == nil || prior == nil || resp.Diagnostics.HasError() {
		return
	}
	keepSmtChainIDs(*planned, *prior)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(smtChainAttr), planned)...)
}

// keepSmtChainIDs fills each planned item's unknown id with the id of the first
// unclaimed prior item of the same type and name.
func keepSmtChainIDs(planned, prior []smtItemModel) {
	used := map[string]bool{}
	for _, item := range planned {
		if !item.ID.IsUnknown() {
			used[item.ID.ValueString()] = true
		}
	}
	for i, item := range planned {
		if !item.ID.IsUnknown() {
			continue
		}
		for _, before := range prior {
			if !used[before.ID.ValueString()] && before.Type.Equal(item.Type) && before.Name.Equal(item.Name) {
				planned[i].ID = before.ID
				used[before.ID.ValueString()] = true
				break
			}
		}
	}
}

func decodeJSONObject(value jsontypes.Normalized) (map[string]any, error) {
	out := map[string]any{}
	if value.IsNull() || value.IsUnknown() {
		return out, nil
	}
	if err := json.Unmarshal([]byte(value.ValueString()), &out); err != nil {
		return nil, fmt.Errorf("must be a JSON object: %w", err)
	}
	if out == nil {
		return nil, errors.New("must be a JSON object, not null")
	}
	return out, nil
}

func stringMap(m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	out := make(map[string]string, len(m.Elements()))
	for key, value := range m.Elements() {
		out[key] = value.(types.String).ValueString()
	}
	return out
}

// smtChainToAPI builds the chain to send. A nil result keeps the stored chain;
// removing smt_chain from the configuration sends [] so the chain goes too.
func smtChainToAPI(planned, prior *[]smtItemModel) (*[]api.SmtChainItem, []string, error) {
	if planned == nil {
		if prior == nil {
			return nil, nil, nil
		}
		return &[]api.SmtChainItem{}, nil, nil
	}
	priorSecrets := map[string]map[string]string{}
	if prior != nil {
		for _, item := range *prior {
			priorSecrets[item.ID.ValueString()] = stringMap(item.SecretConfig)
		}
	}
	var secretValues []string
	out := make([]api.SmtChainItem, 0, len(*planned))
	for i, item := range *planned {
		config, err := decodeJSONObject(item.Config)
		if err != nil {
			return nil, nil, fmt.Errorf("smt_chain[%d].config %w", i, err)
		}
		secrets := stringMap(item.SecretConfig)
		// The API keeps an omitted secret, so one dropped from secret_config is sent as "" to clear it.
		for key := range priorSecrets[item.ID.ValueString()] {
			if _, kept := secrets[key]; !kept && config[key] == nil {
				config[key] = ""
			}
		}
		for key, value := range secrets {
			config[key] = value
			if value != "" {
				secretValues = append(secretValues, value)
			}
		}
		sent := api.SmtChainItem{ID: item.ID.ValueString(), Type: item.Type.ValueString(), Name: item.Name.ValueString(), Enabled: item.Enabled.ValueBool(), Config: config}
		if p := item.Predicate; p != nil {
			predicateConfig, err := decodeJSONObject(p.Config)
			if err != nil {
				return nil, nil, fmt.Errorf("smt_chain[%d].predicate.config %w", i, err)
			}
			sent.Predicate = &api.SmtPredicate{Type: p.Type.ValueString(), Config: predicateConfig, Negate: p.Negate.ValueBool()}
		}
		out = append(out, sent)
	}
	return &out, secretValues, nil
}

// smtChainApplied is the planned chain with the ids the API assigned. The
// response lists the items in the order they were sent.
func smtChainApplied(planned *[]smtItemModel, got *[]api.SmtChainItem) (*[]smtItemModel, error) {
	if planned == nil {
		return nil, nil
	}
	var items []api.SmtChainItem
	if got != nil {
		items = *got
	}
	if len(items) != len(*planned) {
		return nil, fmt.Errorf("sent %d smt_chain items, the API returned %d", len(*planned), len(items))
	}
	applied := slices.Clone(*planned)
	for i := range applied {
		applied[i].ID = types.StringValue(items[i].ID)
	}
	return &applied, nil
}

func encodeJSONObject(config map[string]any, prior jsontypes.Normalized) (jsontypes.Normalized, error) {
	if len(config) == 0 && prior.IsNull() {
		return jsontypes.NewNormalizedNull(), nil
	}
	encoded, err := json.Marshal(config)
	return jsontypes.NewNormalizedValue(string(encoded)), err
}

// smtChainFromAPI converts the chain the API returned. Secrets go to
// secret_config: those prior state holds there, and any isSecret names. A
// masked secret keeps the prior value and is dropped when there is none.
func smtChainFromAPI(got *[]api.SmtChainItem, prior *[]smtItemModel, isSecret func(smtType, key string) bool) (*[]smtItemModel, error) {
	if got == nil || len(*got) == 0 {
		if prior == nil {
			return nil, nil
		}
		return &[]smtItemModel{}, nil
	}
	priorByID := map[string]smtItemModel{}
	if prior != nil {
		for _, item := range *prior {
			priorByID[item.ID.ValueString()] = item
		}
	}
	out := make([]smtItemModel, 0, len(*got))
	for _, item := range *got {
		before := priorByID[item.ID]
		priorConfig, err := decodeJSONObject(before.Config)
		if err != nil {
			return nil, fmt.Errorf("smt_chain item %s config in state %w", item.ID, err)
		}
		priorSecrets := stringMap(before.SecretConfig)
		config, secrets := map[string]any{}, map[string]attr.Value{}
		for key, value := range item.Config {
			secret, isString := value.(string)
			priorSecret, wasSecret := priorSecrets[key]
			priorValue, wasPlain := priorConfig[key]
			switch {
			case isString && strings.HasPrefix(secret, smtSecretMask) && wasSecret:
				secrets[key] = types.StringValue(priorSecret)
			case isString && strings.HasPrefix(secret, smtSecretMask) && wasPlain:
				config[key] = priorValue
			case isString && strings.HasPrefix(secret, smtSecretMask):
				// A secret set outside Terraform: there is no value to keep.
			case isString && (wasSecret || !wasPlain && isSecret(item.Type, key)):
				secrets[key] = types.StringValue(secret)
			default:
				config[key] = value
			}
		}
		// The API HTML-escapes names; state keeps the name as configured.
		model := smtItemModel{ID: types.StringValue(item.ID), Type: types.StringValue(item.Type), Name: types.StringValue(html.UnescapeString(item.Name)), Enabled: types.BoolValue(item.Enabled)}
		if model.Config, err = encodeJSONObject(config, before.Config); err != nil {
			return nil, err
		}
		model.SecretConfig = types.MapNull(types.StringType)
		if len(secrets) > 0 || !before.SecretConfig.IsNull() {
			model.SecretConfig = types.MapValueMust(types.StringType, secrets)
		}
		if p := item.Predicate; p != nil {
			priorPredicate := jsontypes.NewNormalizedNull()
			if before.Predicate != nil {
				priorPredicate = before.Predicate.Config
			}
			predicate := &smtPredicateModel{Type: types.StringValue(p.Type), Negate: types.BoolValue(p.Negate)}
			if predicate.Config, err = encodeJSONObject(p.Config, priorPredicate); err != nil {
				return nil, err
			}
			model.Predicate = predicate
		}
		out = append(out, model)
	}
	return &out, nil
}

// smtSecretNames returns a lookup of the catalog's secret properties, fetched
// on first use. The catalog is empty while chains are off for the service;
// every value is then treated as secret so none reaches plan output.
func (r *BaseConnectorResource) smtSecretNames(ctx context.Context, errOut *error) func(smtType, key string) bool {
	var secret map[string]bool
	return func(smtType, key string) bool {
		if secret == nil {
			secret = map[string]bool{}
			catalog, err := r.client.ListSmtCatalog(ctx, string(r.config.GetConnectorType())+"s")
			if err != nil {
				*errOut = fmt.Errorf("listing the SMT catalog: %w", err)
			}
			for _, entry := range catalog {
				for _, property := range entry.Properties {
					secret[entry.Type+"\x00"+property.Name] = property.Secret
				}
			}
		}
		return len(secret) == 0 || secret[smtType+"\x00"+key]
	}
}

// smtChainRequest reads the planned chain and builds the one to send. state is
// nil on Create. ctx carries the chain's secrets so API errors mask them.
func (r *BaseConnectorResource) smtChainRequest(ctx context.Context, plan, state attrGetter) (context.Context, *[]smtItemModel, *[]api.SmtChainItem, diag.Diagnostics) {
	if !r.hasSmtChain() {
		return ctx, nil, nil, nil
	}
	planned, diags := getSmtChain(ctx, plan)
	var prior *[]smtItemModel
	if state != nil {
		var plannedList, priorList types.List
		diags.Append(plan(ctx, path.Root(smtChainAttr), &plannedList)...)
		diags.Append(state(ctx, path.Root(smtChainAttr), &priorList)...)
		// An unchanged chain is not sent, so other edits still save while chains are off for the service.
		if plannedList.Equal(priorList) {
			return ctx, planned, nil, diags
		}
		var priorDiags diag.Diagnostics
		prior, priorDiags = getSmtChain(ctx, state)
		diags.Append(priorDiags...)
	}
	if diags.HasError() {
		return ctx, nil, nil, diags
	}
	sent, secretValues, err := smtChainToAPI(planned, prior)
	if err != nil {
		diags.AddError("Invalid smt_chain", err.Error())
	}
	return api.WithSentSecrets(ctx, secretValues...), planned, sent, diags
}

// setSmtChainApplied writes the applied chain over what the model put in state.
func (r *BaseConnectorResource) setSmtChainApplied(ctx context.Context, state *tfsdk.State, planned *[]smtItemModel, got *[]api.SmtChainItem) diag.Diagnostics {
	if !r.hasSmtChain() {
		return nil
	}
	applied, err := smtChainApplied(planned, got)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Error saving smt_chain", err.Error())}
	}
	return state.SetAttribute(ctx, path.Root(smtChainAttr), applied)
}

// readSmtChain writes the chain the API returned over what the model put in state.
func (r *BaseConnectorResource) readSmtChain(ctx context.Context, prior attrGetter, state *tfsdk.State, got *[]api.SmtChainItem) diag.Diagnostics {
	if !r.hasSmtChain() {
		return nil
	}
	before, diags := getSmtChain(ctx, prior)
	if diags.HasError() {
		return diags
	}
	var catalogErr error
	chain, err := smtChainFromAPI(got, before, r.smtSecretNames(ctx, &catalogErr))
	if err = errors.Join(err, catalogErr); err != nil {
		diags.AddError("Error reading smt_chain", err.Error())
		return diags
	}
	return append(diags, state.SetAttribute(ctx, path.Root(smtChainAttr), chain)...)
}
