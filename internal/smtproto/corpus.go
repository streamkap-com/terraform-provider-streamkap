// Package smtproto is a feasibility prototype for the nested SMT chain
// representation on connector resources. It is not registered in the
// provider; its tests are the evidence.
package smtproto

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
	var c Corpus
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("LoadCorpus %s: %w", path, err)
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
