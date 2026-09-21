package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// The SMT chain wire contract, as the backend's
// app/services/smt/fixtures/wire_fixtures.json spells it (wire_version 1,
// backend revision 580de048f89e925d53689873c3d6fe64009c1e6e). A copy of
// that fixture lives in testdata and the decode test pins these types to
// it. Resource code depends on SMTChainAPI only.
//
// The fixture publishes the routes for destinations. kind is the path segment
// ("sources" or "destinations"); a kind the backend has not wired answers a
// plain 404 like any other missing route, which is distinct from the
// structured smt_chain_absent a wired route answers for a connector without
// a chain.

// SMTSecretOperation is one write-only operation carried inside the instance
// it targets: "replace" with a value, or "clear".
type SMTSecretOperation struct {
	Pointer   string `json:"pointer"`
	Operation string `json:"operation"`
	Value     string `json:"value,omitempty"`
}

// Operation names of SMTSecretOperation.
const (
	SMTSecretReplace = "replace"
	SMTSecretClear   = "clear"
)

// SMTInstanceWrite is one user instance in a full chain replacement. An
// omitted id creates; an echoed id keeps the server identity. The config is
// the typed child, unwrapped, with secret leaves absent; secret operations
// ride along with the instance. Predicates are not represented yet, so the
// key is never sent.
type SMTInstanceWrite struct {
	ID               string               `json:"id,omitempty"`
	Type             string               `json:"type"`
	Name             string               `json:"name"`
	Enabled          bool                 `json:"enabled"`
	Config           map[string]any       `json:"config"`
	SecretOperations []SMTSecretOperation `json:"secret_operations,omitempty"`
}

// SMTChainWrite is PUT .../smt-chain: a full replacement of the user chain.
// ExpectedRevision is absent on the write that creates the chain and the
// last-read revision otherwise; the server compares it atomically.
type SMTChainWrite struct {
	ExpectedRevision *string            `json:"expected_revision,omitempty"`
	Transforms       []SMTInstanceWrite `json:"transforms"`
}

// SMTInstanceRead is one instance of a chain snapshot. Kind is "typed" for a
// catalog instance and "opaque" for one the catalog cannot describe, in
// which case only id, alias, name, enabled and reason are set. Predicate is
// kept raw: the provider refuses to manage an instance that carries one.
type SMTInstanceRead struct {
	Kind                  string          `json:"kind"`
	ID                    string          `json:"id"`
	Alias                 string          `json:"alias"`
	Type                  string          `json:"type"`
	SchemaVersion         int64           `json:"schema_version"`
	CodecVersion          int64           `json:"codec_version"`
	Name                  string          `json:"name"`
	Enabled               bool            `json:"enabled"`
	Config                map[string]any  `json:"config"`
	ConfiguredSecretPaths []string        `json:"configured_secret_paths"`
	Predicate             json.RawMessage `json:"predicate"`
	Reason                string          `json:"reason,omitempty"`
}

// HasPredicate reports whether the instance carries a predicate.
func (r SMTInstanceRead) HasPredicate() bool {
	return len(r.Predicate) > 0 && string(r.Predicate) != "null"
}

// SMTOrderEntry is one position of the effective chain order.
type SMTOrderEntry struct {
	Origin    string `json:"origin"`
	Reference string `json:"reference"`
}

// SMTChainSnapshot is one chain projection: the saved intent (desired) or
// the last recorded application.
type SMTChainSnapshot struct {
	ChainRevision        string            `json:"chain_revision"`
	DeploymentGeneration int64             `json:"deployment_generation"`
	Transforms           []SMTInstanceRead `json:"transforms"`
	Managed              json.RawMessage   `json:"managed"`
	ManagedOverrides     json.RawMessage   `json:"managed_overrides"`
	Order                []SMTOrderEntry   `json:"order"`
}

// SMTChainRead is GET .../smt-chain and the body of a successful PUT. The
// provider manages the desired chain; neither snapshot is a live worker
// observation.
type SMTChainRead struct {
	Desired           SMTChainSnapshot  `json:"desired"`
	LastApplied       *SMTChainSnapshot `json:"last_applied"`
	ApplicationStatus string            `json:"application_status"`
}

// SMTErrorEnvelope is the detail of every 4xx these routes raise themselves.
type SMTErrorEnvelope struct {
	Code    string               `json:"code"`
	Message string               `json:"message"`
	Issues  []SMTValidationIssue `json:"issues"`
}

// SMTValidationIssue locates one problem in the submitted document. Pointer
// is the runtime data path into the request body (list rows by index), never
// a secret address.
type SMTValidationIssue struct {
	Code       string  `json:"code"`
	Message    string  `json:"message"`
	InstanceID *string `json:"instance_id"`
	RequestKey *string `json:"request_key"`
	Pointer    string  `json:"pointer"`
}

// Error codes the provider reacts to. The four 409 codes are distinct
// situations: a stale expected revision, a create that lost a race to
// another creator, a write without an expected revision against a chain
// that already exists, and a write against a connector whose deletion is
// pending.
const (
	smtRevisionConflictCode  = "smt_revision_conflict"
	smtChainExistsCode       = "smt_chain_exists"
	smtRevisionRequiredCode  = "smt_expected_revision_required"
	smtConnectorDeletingCode = "smt_connector_deleting"
	smtChainAbsentCode       = "smt_chain_absent"
)

// SMTChainAPI is the server surface the connector resources drive the chain
// through.
type SMTChainAPI interface {
	GetSMTChain(ctx context.Context, kind, connectorID string) (*SMTChainRead, error)
	PutSMTChain(ctx context.Context, kind, connectorID string, chain SMTChainWrite) (*SMTChainRead, error)
}

// SMTErrorOf decodes the structured envelope of a chain route error.
func SMTErrorOf(err error) (*SMTErrorEnvelope, bool) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.DetailJSON) == 0 {
		return nil, false
	}
	var env SMTErrorEnvelope
	if json.Unmarshal(apiErr.DetailJSON, &env) != nil || env.Code == "" {
		return nil, false
	}
	return &env, true
}

func smtErrorCode(err error) string {
	env, ok := SMTErrorOf(err)
	if !ok {
		return ""
	}
	return env.Code
}

// IsSMTRevisionConflict reports whether a chain write was refused because the
// chain moved since it was read: the expected revision is stale, or a create
// lost the race to another creator.
func IsSMTRevisionConflict(err error) bool {
	code := smtErrorCode(err)
	return code == smtRevisionConflictCode || code == smtChainExistsCode
}

// IsSMTRevisionRequired reports whether a chain write without an expected
// revision was refused because the connector already has a chain.
func IsSMTRevisionRequired(err error) bool {
	return smtErrorCode(err) == smtRevisionRequiredCode
}

// IsSMTConnectorDeleting reports whether a chain write was refused because
// the connector's deletion is pending: the deletion drops the chain document,
// so nothing written now would survive it.
func IsSMTConnectorDeleting(err error) bool {
	return smtErrorCode(err) == smtConnectorDeletingCode
}

// IsSMTChainAbsent reports whether a chain read answered that the connector
// has no chain yet, which is an empty chain to the provider.
func IsSMTChainAbsent(err error) bool {
	return smtErrorCode(err) == smtChainAbsentCode
}

func smtChainPath(kind, connectorID string) string {
	return "/" + kind + "/" + connectorID + "/smt-chain"
}

func (s *streamkapAPI) GetSMTChain(ctx context.Context, kind, connectorID string) (*SMTChainRead, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+smtChainPath(kind, connectorID), http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp SMTChainRead
	if err := s.doRequest(ctx, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PutSMTChain replaces the chain. The body carries secret values, so it is
// never logged, and it is not retried: a replay of an interrupted write is
// refused as stale by the revision guard, and the outcome of the original
// must surface rather than be papered over.
func (s *streamkapAPI) PutSMTChain(ctx context.Context, kind, connectorID string, chain SMTChainWrite) (*SMTChainRead, error) {
	payload, err := json.Marshal(chain)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+smtChainPath(kind, connectorID), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("PutSMTChain %s %s: %d instances", kind, connectorID, len(chain.Transforms)))
	var resp SMTChainRead
	if err := s.doRequest(ctx, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
