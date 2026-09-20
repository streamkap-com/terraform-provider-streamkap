// Package smt is the nested SMT chain representation on connector resources:
// the typed schema derived from a pinned catalog artifact, the value codec
// between framework values and the wire, key correlation, the write-only
// secret input and the plan/apply logic the connector base resource drives
// when a generated connector opts in.
package smt

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	TypeID        string            `json:"type_id"`
	SchemaVersion int64             `json:"schema_version"`
	CodecVersion  int64             `json:"codec_version"`
	Publishable   bool              `json:"publishable"`
	DataSchema    *JSONSchema       `json:"data_schema"`
	SecretPaths   []string          `json:"secret_paths"`
	RowKeys       map[string]string `json:"row_keys"`
	Examples      struct {
		ReadProjection map[string]any `json:"read_projection"`
	} `json:"examples"`
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
	var c Corpus
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("ParseCorpus: %w", err)
	}
	if c.ContractVersion == 0 {
		return nil, fmt.Errorf("ParseCorpus: contract_version is missing, this is not a catalog artifact")
	}
	sort.Slice(c.Types, func(i, j int) bool { return c.Types[i].TypeID < c.Types[j].TypeID })
	return &c, nil
}

// Catalog is the set of types a connector's chain may reference, by type_id.
type Catalog map[string]*TypeSpec

// NewCatalog indexes the corpus types.
func NewCatalog(c *Corpus) Catalog {
	cat := make(Catalog, len(c.Types))
	for i := range c.Types {
		cat[c.Types[i].TypeID] = &c.Types[i]
	}
	return cat
}

// Publishable returns the catalog without its publishable: false entries. A
// non-publishable type is a contract fixture and must never be offered by the
// generated schema, so both generation and the runtime build from this view.
func (c Catalog) Publishable() Catalog {
	out := make(Catalog, len(c))
	for id, spec := range c {
		if spec.Publishable {
			out[id] = spec
		}
	}
	return out
}

// TypeIDs returns the catalog's type ids sorted, the order schema attributes
// are declared in.
func (c Catalog) TypeIDs() []string {
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
