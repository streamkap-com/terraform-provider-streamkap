package connector

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func smtItem(id, smtType, name, config string, secrets map[string]string) smtItemModel {
	item := smtItemModel{
		ID:           types.StringUnknown(),
		Type:         types.StringValue(smtType),
		Name:         types.StringValue(name),
		Enabled:      types.BoolValue(true),
		Config:       jsontypes.NewNormalizedValue(config),
		SecretConfig: types.MapNull(types.StringType),
	}
	if id != "" {
		item.ID = types.StringValue(id)
	}
	if secrets != nil {
		values := map[string]attr.Value{}
		for key, value := range secrets {
			values[key] = types.StringValue(value)
		}
		item.SecretConfig = types.MapValueMust(types.StringType, values)
	}
	return item
}

func TestSmtChainIDsSurviveReorder(t *testing.T) {
	prior := []smtItemModel{
		smtItem("MaskField1", "mask_field", "hash emails", `{}`, nil),
		smtItem("DropFields1", "drop_fields", "drop notes", `{}`, nil),
	}
	planned := []smtItemModel{
		smtItem("", "drop_fields", "drop notes", `{}`, nil),
		smtItem("", "regex_router", "route", `{}`, nil),
		smtItem("", "mask_field", "hash emails", `{"mask.function":"MD5"}`, nil),
	}
	keepSmtChainIDs(planned, prior)
	require.Equal(t, "DropFields1", planned[0].ID.ValueString())
	require.True(t, planned[1].ID.IsUnknown(), "a new item gets its id from the server")
	require.Equal(t, "MaskField1", planned[2].ID.ValueString())
}

func TestSmtChainToAPI(t *testing.T) {
	prior := []smtItemModel{smtItem("MaskField1", "mask_field", "hash", `{}`, map[string]string{"mask.salt": "old"})}
	planned := []smtItemModel{
		smtItem("", "regex_router", "route", `{"regex":"(.*)","replacement":"x_$1"}`, nil),
		smtItem("MaskField1", "mask_field", "hash", `{"fields.include.list":["t.a","t.b"],"replace.null.with.default":false}`, nil),
	}
	planned[0].Predicate = &smtPredicateModel{Type: types.StringValue("topic_name_matches"), Config: jsontypes.NewNormalizedValue(`{"pattern":"orders"}`), Negate: types.BoolValue(true)}

	sent, _, err := smtChainToAPI(&planned, &prior)
	require.NoError(t, err)
	body, err := json.Marshal(api.Destination{SmtChain: sent})
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"","name":"","connector":"","config":null,"tags":null,"smt_chain":[
		{"type":"regex_router","name":"route","enabled":true,"config":{"regex":"(.*)","replacement":"x_$1"},
		 "predicate":{"type":"topic_name_matches","config":{"pattern":"orders"},"negate":true}},
		{"id":"MaskField1","type":"mask_field","name":"hash","enabled":true,
		 "config":{"fields.include.list":["t.a","t.b"],"replace.null.with.default":false,"mask.salt":""}}]}`, string(body),
		"order and ids are kept, lists and bools keep their JSON types, and a secret removed from secret_config is cleared")

	planned[1].SecretConfig = types.MapValueMust(types.StringType, map[string]attr.Value{"mask.salt": types.StringValue("pepper")})
	sent, secrets, err := smtChainToAPI(&planned, &prior)
	require.NoError(t, err)
	require.Equal(t, "pepper", (*sent)[1].Config["mask.salt"])
	require.Equal(t, []string{"pepper"}, secrets)
}

func TestSmtChainRemovedSendsEmptyList(t *testing.T) {
	prior := []smtItemModel{smtItem("Filter1", "filter", "drop heartbeats", `{}`, nil)}
	sent, _, err := smtChainToAPI(nil, &prior)
	require.NoError(t, err)
	body, err := json.Marshal(api.Source{SmtChain: sent})
	require.NoError(t, err)
	require.Contains(t, string(body), `"smt_chain":[]`)

	sent, _, err = smtChainToAPI(nil, nil)
	require.NoError(t, err)
	body, err = json.Marshal(api.Source{SmtChain: sent})
	require.NoError(t, err)
	require.NotContains(t, string(body), "smt_chain", "a connector that never had a chain keeps the stored one")
}

func TestSmtChainFromAPISecrets(t *testing.T) {
	prior := []smtItemModel{smtItem("MaskField1", "mask_field", "hash", `{"fields.include.list":["t.a"]}`, map[string]string{"mask.salt": "pepper"})}
	got := []api.SmtChainItem{
		{ID: "MaskField1", Type: "mask_field", Name: "hash", Enabled: true, Config: map[string]any{"fields.include.list": []any{"t.a"}, "mask.salt": "*****"}},
		{ID: "MaskField2", Type: "mask_field", Name: "user&#x27;s salt", Enabled: false, Config: map[string]any{"replace.null.with.default": true, "mask.salt": "clear"}},
	}
	isSecret := func(smtType, key string) bool { return smtType == "mask_field" && key == "mask.salt" }

	chain, err := smtChainFromAPI(&got, &prior, isSecret)
	require.NoError(t, err)
	require.Len(t, *chain, 2)
	first, second := (*chain)[0], (*chain)[1]
	require.JSONEq(t, `{"fields.include.list":["t.a"]}`, first.Config.ValueString())
	require.Equal(t, `"pepper"`, first.SecretConfig.Elements()["mask.salt"].String(), "the mask never replaces the configured secret")
	require.JSONEq(t, `{"replace.null.with.default":true}`, second.Config.ValueString())
	require.Equal(t, `"clear"`, second.SecretConfig.Elements()["mask.salt"].String(), "a catalog secret stays out of config")
	require.False(t, second.Enabled.ValueBool())
	require.Equal(t, "user's salt", second.Name.ValueString(), "the API's HTML escaping does not reach state")

	none, err := smtChainFromAPI(nil, nil, isSecret)
	require.NoError(t, err)
	require.Nil(t, none, "no chain stays null")
}
