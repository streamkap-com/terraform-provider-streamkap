package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UnsupportedConstructError is the generation-time diagnostic for a schema
// shape the typed representation cannot carry. It names the construct and the
// schema path so the catalog author can see what to change; the builder never
// falls back to a dynamic type.
type UnsupportedConstructError struct {
	Path      string
	Construct string
	Detail    string
}

func (e *UnsupportedConstructError) Error() string {
	return fmt.Sprintf("unsupported construct %s at %s: %s", e.Construct, e.Path, e.Detail)
}

// ConfigChild is the typed config attribute for one catalog type together
// with the secret leaves that were stripped out of it.
type ConfigChild struct {
	Attribute schema.SingleNestedAttribute
	// SecretTemplates are the schema-level pointers of the stripped writeOnly
	// leaves, in the corpus's `/routes/*/token` form.
	SecretTemplates []string
}

// BuildConfigChild derives the config attribute for one type from its
// data_schema. writeOnly leaves are excluded: they belong to the write-only
// secret input, never to the config child.
func BuildConfigChild(spec *TypeSpec) (*ConfigChild, error) {
	if spec.DataSchema == nil {
		return nil, &UnsupportedConstructError{Path: "/", Construct: "missing data_schema", Detail: spec.TypeID}
	}
	b := &builder{}
	a, err := b.object(spec.DataSchema, "", false)
	if err != nil {
		return nil, err
	}
	sort.Strings(b.secrets)
	// The child is selected by `type`; it is Optional so the other types'
	// children can stay null. It is never Computed: a null child for the
	// selected type is a configuration error, not something to backfill.
	a.Computed = false
	a.Default = nil
	a.Optional = true
	return &ConfigChild{Attribute: a, SecretTemplates: b.secrets}, nil
}

type builder struct {
	secrets []string
}

// scalarType resolves the single non-null JSON type of a node, rejecting
// every polymorphic spelling with a path-bearing error.
func (b *builder) scalarType(node *JSONSchema, p string) (string, bool, error) {
	if node.Ref != "" {
		return "", false, &UnsupportedConstructError{Path: p, Construct: "$ref", Detail: "the contract inlines local refs, so a surviving $ref is a recursive reference"}
	}
	if len(node.OneOf) > 0 {
		return "", false, &UnsupportedConstructError{Path: p, Construct: "oneOf", Detail: "polymorphic union"}
	}
	if node.PrefixItems != nil {
		return "", false, &UnsupportedConstructError{Path: p, Construct: "prefixItems", Detail: "heterogeneous tuple"}
	}
	if len(node.Items) > 0 && node.Items[0] == '[' {
		return "", false, &UnsupportedConstructError{Path: p, Construct: "items[]", Detail: "heterogeneous tuple"}
	}
	nullable := false
	if len(node.AnyOf) > 0 {
		var branches []*JSONSchema
		for _, br := range node.AnyOf {
			if t, _ := br.Type.(string); t == "null" {
				nullable = true
				continue
			}
			branches = append(branches, br)
		}
		if len(branches) != 1 {
			return "", false, &UnsupportedConstructError{Path: p, Construct: "anyOf", Detail: fmt.Sprintf("polymorphic union of %d non-null branches", len(branches))}
		}
		inner, innerNullable, err := b.scalarType(branches[0], p)
		if err != nil {
			return "", false, err
		}
		return inner, nullable || innerNullable, nil
	}
	switch t := node.Type.(type) {
	case string:
		if t == "null" {
			return "", false, &UnsupportedConstructError{Path: p, Construct: "type null", Detail: "a node whose only type is null carries no value"}
		}
		return t, false, nil
	case []any:
		var kinds []string
		for _, k := range t {
			s, _ := k.(string)
			if s == "null" {
				nullable = true
				continue
			}
			kinds = append(kinds, s)
		}
		if len(kinds) != 1 {
			return "", false, &UnsupportedConstructError{Path: p, Construct: "type list", Detail: fmt.Sprintf("polymorphic union %v", kinds)}
		}
		return kinds[0], nullable, nil
	case nil:
		return "", false, &UnsupportedConstructError{Path: p, Construct: "missing type", Detail: "no type, anyOf or $ref"}
	default:
		return "", false, &UnsupportedConstructError{Path: p, Construct: "type", Detail: fmt.Sprintf("unexpected %T", node.Type)}
	}
}

// attribute builds the schema attribute for node. It returns a nil attribute
// for a writeOnly leaf after recording its template.
func (b *builder) attribute(node *JSONSchema, p string, required bool) (schema.Attribute, error) {
	if node.WriteOnly {
		b.secrets = append(b.secrets, p)
		return nil, nil
	}
	kind, nullable, err := b.scalarType(node, p)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "object":
		return b.object(node, p, required)
	case "array":
		return b.array(node, p, required)
	case "string", "integer", "boolean", "number":
		// A nullable scalar is simply Optional: null is its absent form.
		_ = nullable
		return b.scalar(kind, node, p, required)
	default:
		return nil, &UnsupportedConstructError{Path: p, Construct: "type " + kind, Detail: "no typed mapping"}
	}
}

func (b *builder) scalar(kind string, node *JSONSchema, p string, required bool) (schema.Attribute, error) {
	hasDefault := len(node.Default) > 0 && string(node.Default) != "null"
	optional := !required || hasDefault
	computed := hasDefault
	switch kind {
	case "string":
		a := schema.StringAttribute{Required: required && !hasDefault, Optional: optional, Computed: computed}
		if hasDefault {
			var v string
			if err := json.Unmarshal(node.Default, &v); err != nil {
				return nil, &UnsupportedConstructError{Path: p, Construct: "default", Detail: err.Error()}
			}
			a.Default = stringdefault.StaticString(v)
		}
		return a, nil
	case "integer":
		a := schema.Int64Attribute{Required: required && !hasDefault, Optional: optional, Computed: computed}
		if hasDefault {
			var v int64
			if err := json.Unmarshal(node.Default, &v); err != nil {
				return nil, &UnsupportedConstructError{Path: p, Construct: "default", Detail: err.Error()}
			}
			a.Default = int64default.StaticInt64(v)
		}
		return a, nil
	case "number":
		a := schema.Float64Attribute{Required: required && !hasDefault, Optional: optional, Computed: computed}
		if hasDefault {
			var v float64
			if err := json.Unmarshal(node.Default, &v); err != nil {
				return nil, &UnsupportedConstructError{Path: p, Construct: "default", Detail: err.Error()}
			}
			a.Default = float64default.StaticFloat64(v)
		}
		return a, nil
	case "boolean":
		a := schema.BoolAttribute{Required: required && !hasDefault, Optional: optional, Computed: computed}
		if hasDefault {
			var v bool
			if err := json.Unmarshal(node.Default, &v); err != nil {
				return nil, &UnsupportedConstructError{Path: p, Construct: "default", Detail: err.Error()}
			}
			a.Default = booldefault.StaticBool(v)
		}
		return a, nil
	}
	return nil, &UnsupportedConstructError{Path: p, Construct: kind, Detail: "unreachable"}
}

// object maps a properties object to SingleNestedAttribute and a
// properties-less additionalProperties object to a homogeneous map.
func (b *builder) object(node *JSONSchema, p string, required bool) (schema.SingleNestedAttribute, error) {
	if len(node.Properties) == 0 {
		// The homogeneous-map gate: never reached by the corpus, built only
		// through BuildMapAttribute so admitting maps stays an explicit decision.
		return schema.SingleNestedAttribute{}, &UnsupportedConstructError{Path: p, Construct: "object without properties", Detail: "homogeneous maps are built by BuildMapAttribute, not admitted into a config child"}
	}
	req := map[string]bool{}
	for _, r := range node.Required {
		req[r] = true
	}
	names := make([]string, 0, len(node.Properties))
	for n := range node.Properties {
		names = append(names, n)
	}
	sort.Strings(names)
	attrs := map[string]schema.Attribute{}
	for _, n := range names {
		child, err := b.attribute(node.Properties[n], p+"/"+n, req[n])
		if err != nil {
			return schema.SingleNestedAttribute{}, err
		}
		if child != nil {
			attrs[n] = child
		}
	}
	a := schema.SingleNestedAttribute{Attributes: attrs, Required: required, Optional: !required}
	if !required {
		// The backend's read projection materialises every declared default
		// (`output: {prefix: "", ...}` for an omitted output), so the plan must
		// carry the same object or the apply is inconsistent. Only an object
		// whose every child has a default can be defaulted this way.
		if def, ok := objectDefault(attrs); ok {
			a.Computed = true
			a.Default = objectdefault.StaticValue(def)
		}
	}
	return a, nil
}

// BuildMapAttribute builds a homogeneous map from an object whose only shape
// is additionalProperties. Kept apart from the config child so admitting maps
// stays an explicit decision.
func BuildMapAttribute(node *JSONSchema, p string) (schema.Attribute, error) {
	if len(node.AdditionalProperties) == 0 || string(node.AdditionalProperties) == "false" || string(node.AdditionalProperties) == "true" {
		return nil, &UnsupportedConstructError{Path: p, Construct: "additionalProperties", Detail: "a map needs a value schema"}
	}
	var elem JSONSchema
	if err := json.Unmarshal(node.AdditionalProperties, &elem); err != nil {
		return nil, &UnsupportedConstructError{Path: p, Construct: "additionalProperties", Detail: err.Error()}
	}
	b := &builder{}
	child, err := b.attribute(&elem, p+"/*", true)
	if err != nil {
		return nil, err
	}
	if nested, ok := child.(schema.SingleNestedAttribute); ok {
		return schema.MapNestedAttribute{Optional: true, NestedObject: schema.NestedAttributeObject{Attributes: nested.Attributes}}, nil
	}
	return schema.MapAttribute{Optional: true, ElementType: child.GetType()}, nil
}

func (b *builder) array(node *JSONSchema, p string, required bool) (schema.Attribute, error) {
	if len(node.Items) == 0 {
		return nil, &UnsupportedConstructError{Path: p, Construct: "array without items", Detail: "element type unknown"}
	}
	var elem JSONSchema
	if err := json.Unmarshal(node.Items, &elem); err != nil {
		return nil, &UnsupportedConstructError{Path: p, Construct: "items", Detail: err.Error()}
	}
	if elem.WriteOnly {
		return nil, &UnsupportedConstructError{Path: p + "/*", Construct: "writeOnly list element", Detail: "secrets are addressed by row key, a bare secret list has none"}
	}
	child, err := b.attribute(&elem, p+"/*", true)
	if err != nil {
		return nil, err
	}
	switch c := child.(type) {
	case schema.SingleNestedAttribute:
		a := schema.ListNestedAttribute{
			NestedObject: schema.NestedAttributeObject{Attributes: c.Attributes},
			Required:     required,
			Optional:     !required,
		}
		if !required {
			a.Computed = true
			a.Default = listdefault.StaticValue(types.ListValueMust(c.GetType(), nil))
		}
		return a, nil
	case schema.ListAttribute, schema.ListNestedAttribute:
		return nil, &UnsupportedConstructError{Path: p, Construct: "array of arrays", Detail: "no row identity for a nested list"}
	default:
		a := schema.ListAttribute{ElementType: child.GetType(), Required: required, Optional: !required}
		if !required {
			a.Computed = true
			a.Default = listdefault.StaticValue(types.ListValueMust(child.GetType(), nil))
		}
		return a, nil
	}
}

// objectDefault builds the default object of a container whose every child
// carries a default. A Required child, or a plain Optional one such as a
// nullable scalar, means the container has no single default and stays a
// plain Optional attribute.
func objectDefault(attrs map[string]schema.Attribute) (types.Object, bool) {
	attrTypes := map[string]attr.Type{}
	values := map[string]attr.Value{}
	for name, a := range attrs {
		attrTypes[name] = a.GetType()
		v, ok := DefaultValue(a)
		if !ok {
			return types.Object{}, false
		}
		values[name] = v
	}
	return types.ObjectValueMust(attrTypes, values), true
}

// DefaultValue evaluates an attribute's static default outside a plan. Every
// default the builder emits is a Static* value, so the request is irrelevant.
func DefaultValue(a schema.Attribute) (attr.Value, bool) {
	ctx := context.Background()
	switch t := a.(type) {
	case schema.StringAttribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.StringResponse
		t.Default.DefaultString(ctx, defaults.StringRequest{}, &r)
		return r.PlanValue, true
	case schema.Int64Attribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.Int64Response
		t.Default.DefaultInt64(ctx, defaults.Int64Request{}, &r)
		return r.PlanValue, true
	case schema.Float64Attribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.Float64Response
		t.Default.DefaultFloat64(ctx, defaults.Float64Request{}, &r)
		return r.PlanValue, true
	case schema.BoolAttribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.BoolResponse
		t.Default.DefaultBool(ctx, defaults.BoolRequest{}, &r)
		return r.PlanValue, true
	case schema.ListAttribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.ListResponse
		t.Default.DefaultList(ctx, defaults.ListRequest{}, &r)
		return r.PlanValue, true
	case schema.ListNestedAttribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.ListResponse
		t.Default.DefaultList(ctx, defaults.ListRequest{}, &r)
		return r.PlanValue, true
	case schema.SingleNestedAttribute:
		if t.Default == nil {
			return nil, false
		}
		var r defaults.ObjectResponse
		t.Default.DefaultObject(ctx, defaults.ObjectRequest{}, &r)
		return r.PlanValue, true
	}
	return nil, false
}

// DescribePaths lists every attribute path in a config child with its
// framework type, for the "nothing became Dynamic" assertion.
func DescribePaths(a schema.Attribute, prefix string, out map[string]string) {
	out[prefix] = a.GetType().String()
	var children map[string]schema.Attribute
	switch t := a.(type) {
	case schema.SingleNestedAttribute:
		children = t.Attributes
	case schema.ListNestedAttribute:
		children = t.NestedObject.Attributes
	case schema.MapNestedAttribute:
		children = t.NestedObject.Attributes
	}
	names := make([]string, 0, len(children))
	for n := range children {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		DescribePaths(children[n], strings.TrimSuffix(prefix, "/")+"/"+n, out)
	}
}
