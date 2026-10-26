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
// app/services/smt/fixtures/wire_fixtures.json spells it (wire_version 5,
// backend revision aec997750f7fc38044129dad69e7e177670964e0). A copy of
// that fixture lives in testdata and the decode test pins these types to
// it. Resource code depends on SMTChainAPI only.
//
// The fixture publishes the routes for destinations. kind is the path segment
// ("sources" or "destinations"); a kind the backend has not wired answers a
// plain 404 like any other missing route. A wired route reads every
// connector's chain, possibly without user transforms, under a revision.

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
// ride along with the instance. Predicate references use either a retained
// server id or a request key for a new predicate in the same replacement.
type SMTInstanceWrite struct {
	ID               string                  `json:"id,omitempty"`
	Type             string                  `json:"type"`
	Name             string                  `json:"name"`
	Enabled          bool                    `json:"enabled"`
	Config           map[string]any          `json:"config"`
	SecretOperations []SMTSecretOperation    `json:"secret_operations,omitempty"`
	Predicate        *SMTPredicateAttachment `json:"predicate,omitempty"`
}

// SMTPredicateAttachment links a transform to one retained predicate id or
// one new predicate request key. Negate changes the attachment, not the
// immutable predicate record.
type SMTPredicateAttachment struct {
	ID         string `json:"id,omitempty"`
	RequestKey string `json:"request_key,omitempty"`
	Negate     bool   `json:"negate"`
}

// SMTPredicateWrite is either a retained immutable record (ID only) or a new
// typed record addressed by RequestKey for attachments in the same request.
type SMTPredicateWrite struct {
	ID               string               `json:"id,omitempty"`
	RequestKey       string               `json:"request_key,omitempty"`
	Type             string               `json:"type,omitempty"`
	Config           map[string]any       `json:"config,omitempty"`
	SecretOperations []SMTSecretOperation `json:"secret_operations,omitempty"`
}

// SMTChainWrite is PUT .../smt-chain and the smt_chain a connector create or
// source update embeds. Transforms and managed overrides are independent
// replace-if-present fields. ExpectedRevision is the last-read revision, which
// the server compares atomically; every write to an existing connector
// carries it, and only a connector create, which has nothing to compare
// against, omits it.
type SMTCatalogPrecondition struct {
	ContractVersion int    `json:"contract_version"`
	Digest          string `json:"digest"`
}

type SMTChainWrite struct {
	CatalogPrecondition SMTCatalogPrecondition `json:"catalog_precondition"`
	ExpectedRevision    *string                `json:"expected_revision,omitempty"`
	Predicates          *[]SMTPredicateWrite   `json:"predicates,omitempty"`
	Transforms          *[]SMTInstanceWrite    `json:"transforms,omitempty"`
	ManagedOverrides    *map[string]bool       `json:"managed_overrides,omitempty"`
}

// SMTPredicateRead is one immutable typed predicate record in a chain snapshot.
type SMTPredicateRead struct {
	ID                    string         `json:"id"`
	Alias                 string         `json:"alias"`
	Type                  string         `json:"type"`
	SchemaVersion         int64          `json:"schema_version"`
	CodecVersion          int64          `json:"codec_version"`
	Config                map[string]any `json:"config"`
	ConfiguredSecretPaths []string       `json:"configured_secret_paths"`
}

// SMTInstanceRead is one instance of a chain snapshot. Kind is "typed" for a catalog instance and "opaque" for one the catalog cannot describe, in which case only id, alias, name, enabled and reason are set. The id is the instance's transform alias. Predicate stays raw so the SMT surface can resolve its server id to the provider predicate key.
//
// A selector is write-only: the server compiles it into the field lists of config and reads back those lists with no selector. The read schema still declares the field, so Selector stays raw and an instance that carries one is refused rather than managed, since writing the compiled lists back would drop the rule.
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
	Selector              json.RawMessage `json:"selector"`
	Predicate             json.RawMessage `json:"predicate"`
	Reason                string          `json:"reason,omitempty"`
}

// HasPredicate reports whether the instance carries a predicate.
func (r SMTInstanceRead) HasPredicate() bool {
	return len(r.Predicate) > 0 && string(r.Predicate) != "null"
}

// HasSelector reports whether the instance carries a selector.
func (r SMTInstanceRead) HasSelector() bool {
	return len(r.Selector) > 0 && string(r.Selector) != "null"
}

// SMTOrderEntry is one position of the effective chain order.
type SMTOrderEntry struct {
	Origin    string `json:"origin"`
	Reference string `json:"reference"`
}

// SMTChainSnapshot is one chain projection: the saved intent (desired) or
// the last recorded application.
type SMTChainSnapshot struct {
	ChainRevision        string             `json:"chain_revision"`
	DeploymentGeneration int64              `json:"deployment_generation"`
	Predicates           []SMTPredicateRead `json:"predicates"`
	Transforms           []SMTInstanceRead  `json:"transforms"`
	Managed              json.RawMessage    `json:"managed"`
	ManagedOverrides     map[string]bool    `json:"managed_overrides"`
	Order                []SMTOrderEntry    `json:"order"`
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

// Error codes the provider reacts to. A write without expected_revision is
// answered with the revision conflict code and names the missing revision in
// its issue; smt_read_failed is a 503 whose write may or may not have landed.
const (
	smtRevisionConflictCode         = "smt_revision_conflict"
	smtExpectedRevisionRequiredCode = "smt_expected_revision_required"
	smtConnectorDeletingCode        = "smt_connector_deleting"
	smtReadFailedCode               = "smt_read_failed"
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
// chain moved since it was read: the expected revision is stale.
func IsSMTRevisionConflict(err error) bool {
	env, ok := SMTErrorOf(err)
	return ok && env.Code == smtRevisionConflictCode && !env.hasIssue(smtExpectedRevisionRequiredCode)
}

// IsSMTExpectedRevisionMissing reports whether a write to an existing
// connector was refused because it carried no expected revision.
func IsSMTExpectedRevisionMissing(err error) bool {
	env, ok := SMTErrorOf(err)
	return ok && env.Code == smtRevisionConflictCode && env.hasIssue(smtExpectedRevisionRequiredCode)
}

// IsSMTReadFailed reports whether the chain could not be read. On a write the
// server answers it both when loading the connector failed before saving and
// when the saved chain could not be read back, so the outcome is unknown.
func IsSMTReadFailed(err error) bool {
	return smtErrorCode(err) == smtReadFailedCode
}

func (e *SMTErrorEnvelope) hasIssue(code string) bool {
	for _, issue := range e.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

// IsSMTConnectorDeleting reports whether a chain write was refused because
// the connector's deletion is pending, so nothing written now would survive
// it.
func IsSMTConnectorDeleting(err error) bool {
	return smtErrorCode(err) == smtConnectorDeletingCode
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
	instanceCount := 0
	if chain.Transforms != nil {
		instanceCount = len(*chain.Transforms)
	}
	tflog.Debug(ctx, fmt.Sprintf("PutSMTChain %s %s: %d instances", kind, connectorID, instanceCount))
	var resp SMTChainRead
	if err := s.doRequest(ctx, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
