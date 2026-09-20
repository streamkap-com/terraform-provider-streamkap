package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

// SMTChainConnector names one connector whose generated model carries the
// nested SMT chain. The chain schema itself is built from the pinned catalog
// artifact, never from the connector's plugin config.
type SMTChainConnector struct {
	Connector  string `json:"connector"`
	EntityType string `json:"entity_type"`
}

// smtOptions are the catalog inputs of one generation run. Both are recorded
// in the generated provenance so the schema authority is never a mutable
// checkout.
type smtOptions struct {
	// CatalogPath is the immutable catalog artifact. Required when any
	// connector in the run opts into the chain.
	CatalogPath string
	// BackendRevision is the backend commit the artifact was copied from. The
	// artifact deliberately carries no revision of its own.
	BackendRevision string
}

// smtCatalogProvenance is what the generated catalog file records.
type smtCatalogProvenance struct {
	ArtifactPath     string
	ArtifactSHA256   string
	EnvelopeDigest   string
	BackendRevision  string
	PublishableTypes int
	TotalTypes       int
	TypeIDs          []string
}

// hasSMTChain reports whether the connector opts into the chain.
func (o *OverrideConfig) hasSMTChain(entityType, connectorCode string) bool {
	if o == nil {
		return false
	}
	for _, c := range o.SMTChainConnectors {
		if c.Connector == connectorCode && c.EntityType == entityType+"s" {
			return true
		}
	}
	return false
}

// loadSMTCatalog reads and validates the artifact: it must parse, and every
// publishable type must build into a typed config child. An unsupported
// construct fails here with the type and the schema path, before any file is
// written. The artifact bytes are returned verbatim for the embedded copy.
func loadSMTCatalog(opts smtOptions) ([]byte, *smtCatalogProvenance, error) {
	if opts.CatalogPath == "" {
		return nil, nil, fmt.Errorf("--smt-catalog is required: a connector opts into the SMT chain, and the chain schema is built from the pinned catalog artifact")
	}
	if opts.BackendRevision == "" {
		return nil, nil, fmt.Errorf("--backend-revision is required with --smt-catalog: the artifact carries no revision of its own, so the run must record the backend commit it was copied from")
	}
	raw, err := os.ReadFile(opts.CatalogPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read SMT catalog %s: %w", opts.CatalogPath, err)
	}
	corpus, err := smt.ParseCorpus(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("SMT catalog %s: %w", opts.CatalogPath, err)
	}
	publishable := smt.NewCatalog(corpus).Publishable()
	if _, err := smt.BuildChain(publishable); err != nil {
		return nil, nil, fmt.Errorf("SMT catalog %s: %w", opts.CatalogPath, err)
	}
	sum := sha256.Sum256(raw)
	prov := &smtCatalogProvenance{
		ArtifactPath:     filepath.Base(opts.CatalogPath),
		ArtifactSHA256:   hex.EncodeToString(sum[:]),
		EnvelopeDigest:   corpus.Digest,
		BackendRevision:  opts.BackendRevision,
		PublishableTypes: len(publishable),
		TotalTypes:       len(corpus.Types),
		TypeIDs:          publishable.TypeIDs(),
	}
	if prov.PublishableTypes == 0 {
		fmt.Printf("Warning: SMT catalog %s has no publishable types; smt_chain admits no type until the catalog publishes one\n", opts.CatalogPath)
	}
	return raw, prov, nil
}

// writeSMTCatalog writes the byte-identical artifact copy and the Go file
// that embeds it with its provenance.
func writeSMTCatalog(outputDir string, raw []byte, prov *smtCatalogProvenance) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}
	// #nosec G306 -- a committed artifact copy, like the generated sources.
	if err := os.WriteFile(filepath.Join(outputDir, smtCatalogJSONName), raw, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", smtCatalogJSONName, err)
	}
	tmpl, err := template.New("smt_catalog").Parse(smtCatalogTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse SMT catalog template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, prov); err != nil {
		return fmt.Errorf("failed to execute SMT catalog template: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format %s: %w", smtCatalogGoName, err)
	}
	// #nosec G306 -- committed generated source.
	if err := os.WriteFile(filepath.Join(outputDir, smtCatalogGoName), formatted, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", smtCatalogGoName, err)
	}
	return nil
}

const (
	smtCatalogJSONName = "smt_catalog.json"
	smtCatalogGoName   = "smt_catalog.go"
)

// smtCatalogTemplate renders the embedded catalog and its provenance. The
// runtime builds the chain schema with the same smt.BuildChain call that
// validated the artifact at generation time.
const smtCatalogTemplate = `// Code generated by tfgen. DO NOT EDIT.
//
// SMT catalog provenance. Every connector that opts into smt_chain builds its
// chain schema from this artifact copy, which is byte-identical to the input.
//
//	artifact:          {{ .ArtifactPath }}
//	artifact sha256:   {{ .ArtifactSHA256 }}
//	envelope digest:   {{ .EnvelopeDigest }}
//	backend revision:  {{ .BackendRevision }}
//	publishable types: {{ .PublishableTypes }} of {{ .TotalTypes }}{{ range .TypeIDs }}
//	  - {{ . }}{{ end }}

package generated

import (
	_ "embed"
	"sync"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

//go:embed smt_catalog.json
var smtCatalogJSON []byte

// SMTCatalogSHA256 is the sha256 of the embedded artifact copy.
const SMTCatalogSHA256 = "{{ .ArtifactSHA256 }}"

// SMTCatalogBackendRevision is the backend commit the artifact was copied from.
const SMTCatalogBackendRevision = "{{ .BackendRevision }}"

var smtChain = sync.OnceValues(func() (*smt.ChainSchema, error) {
	corpus, err := smt.ParseCorpus(smtCatalogJSON)
	if err != nil {
		return nil, err
	}
	return smt.BuildChain(smt.NewCatalog(corpus).Publishable())
})

// SMTChain returns the chain schema built from the pinned catalog. The same
// builder accepted the same bytes at generation time, so a failure here is a
// broken build, not a configuration error.
func SMTChain() *smt.ChainSchema {
	chain, err := smtChain()
	if err != nil {
		panic("internal/generated: embedded SMT catalog does not build: " + err.Error())
	}
	return chain
}
`
