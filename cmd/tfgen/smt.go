package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"slices"
	"text/template"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

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
	ArtifactPath    string
	ArtifactSHA256  string
	EnvelopeDigest  string
	BackendRevision string
	TotalTypes      int
	// Hosts are the catalog hosts that publish at least one type, sorted. Each
	// builds its own chain schema from its authorable types.
	Hosts []smtHostProvenance
}

// smtHostProvenance is one host's view of the catalog.
type smtHostProvenance struct {
	Host    string
	TypeIDs []string
}

// chainHosts are the hosts whose connectors can carry a chain, in the
// backend's wire spelling.
var chainHosts = []string{"destinations", "sources"}

// hostSet is the set of hosts with a chain schema in this run.
func (p *smtCatalogProvenance) hostSet() map[string]bool {
	if p == nil {
		return nil
	}
	set := make(map[string]bool, len(p.Hosts))
	for _, h := range p.Hosts {
		set[h.Host] = true
	}
	return set
}

// smtChainHost is the catalog host whose chain the connector's model carries,
// or "" when it carries none. It mirrors the backend's potential_support rule
// (composition_context.potentially_supported, which gates the chain routes):
// a source or destination other than kafkadirect or an API source, on a host
// for which the pinned catalog publishes at least one type.
func (g *Generator) smtChainHost(config *ConnectorConfig, connectorCode string) string {
	if connectorCode == "kafkadirect" || config.APISource || (g.entityType != "source" && g.entityType != "destination") {
		return ""
	}
	host := g.entityType + "s"
	if !g.smtHosts[host] {
		return ""
	}
	return host
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
	if corpus.ContractVersion != 1 {
		return nil, nil, fmt.Errorf("SMT catalog %s: unsupported contract_version %d, want 1", opts.CatalogPath, corpus.ContractVersion)
	}
	fullCatalog := smt.NewCatalog(corpus)
	sum := sha256.Sum256(raw)
	prov := &smtCatalogProvenance{
		ArtifactPath:    filepath.Base(opts.CatalogPath),
		ArtifactSHA256:  hex.EncodeToString(sum[:]),
		EnvelopeDigest:  corpus.Digest,
		BackendRevision: opts.BackendRevision,
		TotalTypes:      len(corpus.Types),
	}
	for _, host := range chainHosts {
		if !publishesFor(corpus, host) {
			continue
		}
		authorable := fullCatalog.ForHost(host).Authorable()
		if _, err := smt.BuildChain(authorable); err != nil {
			return nil, nil, fmt.Errorf("SMT catalog %s: %s: %w", opts.CatalogPath, host, err)
		}
		if authorable.Len() == 0 {
			fmt.Printf("Warning: SMT catalog %s publishes no authorable %s type; their smt_chain admits no new instance until it does\n", opts.CatalogPath, host)
		}
		prov.Hosts = append(prov.Hosts, smtHostProvenance{Host: host, TypeIDs: authorable.TypeIDs()})
	}
	return raw, prov, nil
}

// publishesFor reports whether the catalog publishes any type, authorable or
// withdrawn, for host: the backend lists both in a connector's catalog, so
// either makes the connector potentially supported.
func publishesFor(corpus *smt.Corpus, host string) bool {
	for i := range corpus.Types {
		if corpus.Types[i].Publishable && slices.Contains(corpus.Types[i].Hosts, host) {
			return true
		}
	}
	return false
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
// runtime builds each host's chain schema with the same smt.BuildChain call
// that validated the artifact at generation time.
const smtCatalogTemplate = `// Code generated by tfgen. DO NOT EDIT.
//
// SMT catalog provenance. Every connector model that carries smt_chain builds
// its chain schema from this artifact copy, which is byte-identical to the
// input, for its own host.
//
//	artifact:          {{ .ArtifactPath }}
//	artifact sha256:   {{ .ArtifactSHA256 }}
//	envelope digest:   {{ .EnvelopeDigest }}
//	backend revision:  {{ .BackendRevision }}
//	catalog types:     {{ .TotalTypes }}{{ range .Hosts }}
//	{{ .Host }} authorable types: {{ len .TypeIDs }}{{ range .TypeIDs }}
//	  - {{ . }}{{ end }}{{ end }}

package generated

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

//go:embed smt_catalog.json
var smtCatalogJSON []byte

// SMTCatalogSHA256 is the sha256 of the embedded artifact copy.
const SMTCatalogSHA256 = "{{ .ArtifactSHA256 }}"

// SMTCatalogBackendRevision is the backend commit the artifact was copied from.
const SMTCatalogBackendRevision = "{{ .BackendRevision }}"

// smtChains builds each host's chain once. A host is listed when the catalog
// publishes at least one type for it.
var smtChains = map[string]func() (*smt.ChainSchema, error){
{{- range .Hosts }}
	"{{ .Host }}": sync.OnceValues(func() (*smt.ChainSchema, error) { return buildSMTChain("{{ .Host }}") }),
{{- end }}
}

func buildSMTChain(host string) (*smt.ChainSchema, error) {
	corpus, err := smt.ParseCorpus(smtCatalogJSON)
	if err != nil {
		return nil, err
	}
	return smt.BuildChain(smt.NewCatalog(corpus).ForHost(host).Authorable())
}

// SMTChain returns the chain schema for host ("destinations" or "sources")
// built from the pinned catalog. The same builder accepted the same bytes at
// generation time, and only generated models name a host, so a failure here is
// a broken build, not a configuration error.
func SMTChain(host string) *smt.ChainSchema {
	build, ok := smtChains[host]
	if !ok {
		panic(fmt.Sprintf("internal/generated: the embedded SMT catalog has no chain for host %q", host))
	}
	chain, err := build()
	if err != nil {
		panic("internal/generated: embedded SMT catalog does not build: " + err.Error())
	}
	return chain
}
`
