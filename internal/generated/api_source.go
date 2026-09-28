package generated

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
	// ClientFields name the config fields holding the customer's own OAuth
	// client, which the connect flow's start request must carry.
	ClientFields []string
}
