package generated

import "github.com/hashicorp/terraform-plugin-framework/resource/schema"

// APISource is what tfgen generates for one API source: its schema, model
// and the form rules the provider checks at plan time.
type APISource struct {
	Code          string
	DisplayName   string
	Schema        func() schema.Schema
	FieldMappings map[string]string
	NewModel      func() any
	Conditions    []APICondition
	Dependencies  []APIDependency
	OAuth         APIOAuth
}

// APICondition is an API-source field the form shows only while
// ConditionField holds one of ConditionValues; ConditionDefault applies when
// it is unset. A Required field must be set whenever it is shown.
type APICondition struct {
	Field            string
	ConditionField   string
	ConditionValues  []string
	ConditionDefault string
	Required         bool
}

// APIDependency is a field that, when set, needs every field in Requires set
// too.
type APIDependency struct {
	Field    string
	Requires []string
}

// APIOAuth describes an API source's Connect (OAuth) flow. An enabled flow
// with an empty AuthModeField is the vendor's only way to sign in.
type APIOAuth struct {
	Enabled         bool
	AuthModeField   string
	AuthModeValue   string
	AuthModeDefault string
	// ConnectCommand is the CLI command that starts Connect, with the
	// options this vendor's flow needs.
	ConnectCommand string
	// HostField names the config field the grant is authorized for (the
	// Zendesk subdomain): changing it needs a new grant.
	HostField string
}
