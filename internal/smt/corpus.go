// Package smt is the nested SMT chain representation on connector resources:
// the typed schema derived from a pinned catalog artifact, the value codec
// between framework values and the wire, key correlation, the write-only
// secret input and the plan/apply logic the connector base resource drives
// when a generated connector opts in.
package smt

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Corpus is the backend-produced contract artifact. Only the envelope keys
// the provider consumes are decoded; ui_schema is JSON Forms and stays opaque.
type Corpus struct {
	ContractVersion int        `json:"contract_version"`
	Producer        string     `json:"producer"`
	Digest          string     `json:"digest"`
	Types           []TypeSpec `json:"types"`
}

// TypeSpec is one catalog type: the pinned identity a persisted instance
// references plus the shapes the provider derives its schema from.
type TypeSpec struct {
	TypeID                   string               `json:"type_id"`
	SchemaVersion            int64                `json:"schema_version"`
	CodecVersion             int64                `json:"codec_version"`
	Publishable              bool                 `json:"publishable"`
	Authorable               bool                 `json:"authorable"`
	Role                     string               `json:"role"`
	Hosts                    []string             `json:"hosts"`
	CompatibleTransformTypes []string             `json:"compatible_transform_types"`
	SupportsNegate           bool                 `json:"supports_negate"`
	DataSchema               *JSONSchema          `json:"data_schema"`
	UISchema                 json.RawMessage      `json:"ui_schema"`
	ConsumerRequirements     ConsumerRequirements `json:"consumer_requirements"`
	SecretPaths              []string             `json:"secret_paths"`
	RowKeys                  map[string]string    `json:"row_keys"`
	Examples                 struct {
		ReadProjection map[string]any `json:"read_projection"`
	} `json:"examples"`
	schemaSemantics string
}

// JSONSchema is the subset of JSON Schema the contract promises: a
// self-contained tree with no $ref resolver. Fields the provider must reject
// (Ref, OneOf, PrefixItems, a tuple Items) are decoded so the rejection can
// name them.
type JSONSchema struct {
	Type                 any                    `json:"type"`
	Properties           map[string]*JSONSchema `json:"properties"`
	Required             []string               `json:"required"`
	Items                json.RawMessage        `json:"items"`
	PrefixItems          json.RawMessage        `json:"prefixItems"`
	AdditionalProperties json.RawMessage        `json:"additionalProperties"`
	AnyOf                []*JSONSchema          `json:"anyOf"`
	OneOf                []*JSONSchema          `json:"oneOf"`
	Ref                  string                 `json:"$ref"`
	Default              json.RawMessage        `json:"default"`
	WriteOnly            bool                   `json:"writeOnly"`
	Title                string                 `json:"title"`
}

// LoadCorpus reads the artifact copy and returns its types in a stable order.
func LoadCorpus(path string) (*Corpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadCorpus: %w", err)
	}
	c, err := ParseCorpus(raw)
	if err != nil {
		return nil, fmt.Errorf("LoadCorpus %s: %w", path, err)
	}
	return c, nil
}

// ParseCorpus decodes the artifact bytes and returns its types in a stable
// order.
func ParseCorpus(raw []byte) (*Corpus, error) {
	var source struct {
		Types []json.RawMessage `json:"types"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, fmt.Errorf("ParseCorpus requirements: %w", err)
	}
	for i, rawType := range source.Types {
		if err := validateRawNumericType(rawType); err != nil {
			return nil, fmt.Errorf("ParseCorpus: types[%d]: %w", i, err)
		}
		if err := validateRawAuthorable(rawType); err != nil {
			return nil, fmt.Errorf("ParseCorpus: types[%d]: %w", i, err)
		}
	}
	var c Corpus
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("ParseCorpus: %w", err)
	}
	if c.ContractVersion == 0 {
		return nil, fmt.Errorf("ParseCorpus: contract_version is missing, this is not a catalog artifact")
	}
	if len(source.Types) != len(c.Types) {
		return nil, fmt.Errorf("ParseCorpus: decoded %d types but retained %d raw declarations", len(c.Types), len(source.Types))
	}
	seenPins := make(map[CatalogPin]int, len(c.Types))
	for i := range c.Types {
		spec := &c.Types[i]
		if spec.Role != "transform" && spec.Role != "predicate" {
			return nil, fmt.Errorf("ParseCorpus: types[%d].role must be transform or predicate, got %q", i, spec.Role)
		}
		pin := spec.Pin()
		if first, duplicate := seenPins[pin]; duplicate {
			return nil, fmt.Errorf("ParseCorpus: types[%d]: duplicate catalog pin %s/%s schema %d codec %d (first declared at types[%d])", i, pin.Role, pin.TypeID, pin.SchemaVersion, pin.CodecVersion, first)
		}
		seenPins[pin] = i
		if err := validateConsumerRequirements(source.Types[i], spec); err != nil {
			return nil, fmt.Errorf("ParseCorpus: type %q: %w", spec.TypeID, err)
		}
		if err := validateSecretMetadata(source.Types[i], spec); err != nil {
			return nil, fmt.Errorf("ParseCorpus: type %q: %w", spec.TypeID, err)
		}
		semantics, err := schemaSemantics(source.Types[i])
		if err != nil {
			return nil, fmt.Errorf("ParseCorpus: type %q: %w", spec.TypeID, err)
		}
		spec.schemaSemantics = semantics
	}
	sort.Slice(c.Types, func(i, j int) bool { return c.Types[i].TypeID < c.Types[j].TypeID })
	return &c, nil
}

func validateRawAuthorable(rawType json.RawMessage) error {
	var declaration struct {
		Authorable *bool `json:"authorable"`
	}
	if err := json.Unmarshal(rawType, &declaration); err != nil {
		return fmt.Errorf("authorable must be boolean: %w", err)
	}
	if declaration.Authorable == nil {
		return fmt.Errorf("authorable is required")
	}
	return nil
}

func schemaSemantics(rawType json.RawMessage) (string, error) {
	var declaration struct {
		DataSchema json.RawMessage `json:"data_schema"`
	}
	if err := json.Unmarshal(rawType, &declaration); err != nil {
		return "", fmt.Errorf("decode data_schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(declaration.DataSchema))
	decoder.UseNumber()
	var schema any
	if err := decoder.Decode(&schema); err != nil {
		return "", fmt.Errorf("decode data_schema semantics: %w", err)
	}
	encoded, err := json.Marshal(canonicalSchemaSemantics(schema, ""))
	if err != nil {
		return "", fmt.Errorf("encode data_schema semantics: %w", err)
	}
	return string(encoded), nil
}

func canonicalSchemaSemantics(value any, parentKey string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		secretProperties := map[string]bool{}
		if properties, ok := typed["properties"].(map[string]any); ok {
			for name, property := range properties {
				if declaration, ok := property.(map[string]any); ok && declaration["writeOnly"] == true {
					secretProperties[name] = true
				}
			}
		}
		for key, child := range typed {
			switch key {
			case "title", "description", "examples", "deprecated":
				continue
			}
			if key == "properties" && len(secretProperties) > 0 {
				if properties, ok := child.(map[string]any); ok {
					visible := make(map[string]any, len(properties)-len(secretProperties))
					for name, property := range properties {
						if !secretProperties[name] {
							visible[name] = property
						}
					}
					child = visible
				}
			}
			if key == "required" && len(secretProperties) > 0 {
				if required, ok := child.([]any); ok {
					visible := make([]any, 0, len(required))
					for _, name := range required {
						if stringName, ok := name.(string); !ok || !secretProperties[stringName] {
							visible = append(visible, name)
						}
					}
					child = visible
				}
			}
			out[key] = canonicalSchemaSemantics(child, key)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = canonicalSchemaSemantics(child, parentKey)
		}
		if parentKey == "required" || parentKey == "enum" || parentKey == "anyOf" || parentKey == "type" {
			sort.Slice(out, func(i, j int) bool {
				left, _ := json.Marshal(out[i])
				right, _ := json.Marshal(out[j])
				return string(left) < string(right)
			})
		}
		return out
	default:
		return value
	}
}

// CatalogPin is the complete immutable identity of one released definition.
// Role is part of the key because transform and predicate authoring are
// separate namespaces even when a producer reuses a type id.
type CatalogPin struct {
	Role          string
	TypeID        string
	SchemaVersion int64
	CodecVersion  int64
}

type catalogRoleType struct {
	Role   string
	TypeID string
}

// Catalog retains every pinned definition while recording the producer's
// deterministic current selection separately. Filters never recompute that
// selection: if the current release is ineligible for a host, an older
// release cannot silently become authorable there.
type Catalog struct {
	entries         map[CatalogPin]*TypeSpec
	current         map[catalogRoleType]CatalogPin
	contractVersion int
	digest          string
}

func (s *TypeSpec) Pin() CatalogPin {
	return CatalogPin{Role: s.Role, TypeID: s.TypeID, SchemaVersion: s.SchemaVersion, CodecVersion: s.CodecVersion}
}

// NewCatalog indexes all corpus releases and selects the same current pin as
// the backend: a public authorable release wins, then a public withdrawn
// release, then an internal declaration; schema/codec breaks each tie.
func NewCatalog(c *Corpus) Catalog {
	cat := Catalog{
		entries:         make(map[CatalogPin]*TypeSpec, len(c.Types)),
		current:         make(map[catalogRoleType]CatalogPin),
		contractVersion: c.ContractVersion,
		digest:          c.Digest,
	}
	for i := range c.Types {
		spec := &c.Types[i]
		pin := spec.Pin()
		cat.entries[pin] = spec
		roleType := catalogRoleType{Role: spec.Role, TypeID: spec.TypeID}
		selected, ok := cat.current[roleType]
		if !ok || newerCurrent(spec, cat.entries[selected]) {
			cat.current[roleType] = pin
		}
	}
	return cat
}

func newerCurrent(candidate, selected *TypeSpec) bool {
	candidateSelectable := candidate.Publishable && candidate.Authorable
	selectedSelectable := selected.Publishable && selected.Authorable
	if candidateSelectable != selectedSelectable {
		return candidateSelectable
	}
	if candidate.Publishable != selected.Publishable {
		return candidate.Publishable
	}
	if candidate.SchemaVersion != selected.SchemaVersion {
		return candidate.SchemaVersion > selected.SchemaVersion
	}
	return candidate.CodecVersion > selected.CodecVersion
}

// Lookup returns the exact released artifact identified by all four pin
// fields. It never falls back to another role or version.
func (c Catalog) Lookup(pin CatalogPin) (*TypeSpec, bool) {
	spec, ok := c.entries[pin]
	return spec, ok
}

// Current returns the producer-selected definition for new authoring when
// that exact pin survived the catalog's filters.
func (c Catalog) Current(role, typeID string) (*TypeSpec, bool) {
	pin, ok := c.current[catalogRoleType{Role: role, TypeID: typeID}]
	if !ok {
		return nil, false
	}
	return c.Lookup(pin)
}

// ReleaseIdentity returns the immutable envelope identity retained through
// host and role filters. It is the source for write preconditions once the
// corresponding wire contract is available.
func (c Catalog) ReleaseIdentity() (contractVersion int, digest string) {
	return c.contractVersion, c.digest
}

func (c Catalog) WritePrecondition() (contractVersion int, digest string, err error) {
	if c.contractVersion != 1 {
		return 0, "", fmt.Errorf("unsupported catalog contract_version %d", c.contractVersion)
	}
	const prefix = "sha256:"
	if len(c.digest) != len(prefix)+64 || c.digest[:len(prefix)] != prefix {
		return 0, "", fmt.Errorf("catalog digest must be sha256 followed by 64 lowercase hexadecimal characters")
	}
	encoded := c.digest[len(prefix):]
	if _, decodeErr := hex.DecodeString(encoded); decodeErr != nil || encoded != strings.ToLower(encoded) {
		return 0, "", fmt.Errorf("catalog digest must be sha256 followed by 64 lowercase hexadecimal characters")
	}
	return c.contractVersion, c.digest, nil
}

// Specs returns every retained definition in deterministic identity order.
func (c Catalog) Specs() []*TypeSpec {
	out := make([]*TypeSpec, 0, len(c.entries))
	for _, spec := range c.entries {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		if out[i].TypeID != out[j].TypeID {
			return out[i].TypeID < out[j].TypeID
		}
		if out[i].SchemaVersion != out[j].SchemaVersion {
			return out[i].SchemaVersion < out[j].SchemaVersion
		}
		return out[i].CodecVersion < out[j].CodecVersion
	})
	return out
}

func (c Catalog) filter(keep func(*TypeSpec) bool) Catalog {
	out := Catalog{
		entries:         make(map[CatalogPin]*TypeSpec, len(c.entries)),
		current:         make(map[catalogRoleType]CatalogPin, len(c.current)),
		contractVersion: c.contractVersion,
		digest:          c.digest,
	}
	for pin, spec := range c.entries {
		if keep(spec) {
			out.entries[pin] = spec
		}
	}
	for roleType, pin := range c.current {
		out.current[roleType] = pin
	}
	return out
}

// Publishable returns the catalog entries the transform authoring surface may
// offer. Contract fixtures and predicate records are not transform instances,
// so both generation and runtime build the transform chain from this view.
func (c Catalog) Publishable() Catalog {
	return c.filter(func(spec *TypeSpec) bool {
		return spec.Publishable && spec.Authorable && spec.Role == "transform"
	})
}

// Authorable returns every selectable public role while preserving the role metadata
// BuildChain uses to place each type on the correct authoring surface.
func (c Catalog) Authorable() Catalog {
	return c.filter(func(spec *TypeSpec) bool { return spec.Publishable && spec.Authorable })
}

// ForHost returns only types whose catalog declaration names the exact host.
func (c Catalog) ForHost(host string) Catalog {
	return c.filter(func(spec *TypeSpec) bool {
		for _, declared := range spec.Hosts {
			if declared == host {
				return true
			}
		}
		return false
	})
}

// ForRole retains all exact pins of one authoring namespace.
func (c Catalog) ForRole(role string) Catalog {
	return c.filter(func(spec *TypeSpec) bool { return spec.Role == role })
}

// TypeIDs returns the catalog's type ids sorted, the order schema attributes
// are declared in.
func (c Catalog) TypeIDs() []string {
	set := map[string]bool{}
	for roleType, pin := range c.current {
		if _, ok := c.entries[pin]; ok {
			set[roleType.TypeID] = true
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (c Catalog) Len() int { return len(c.entries) }
