package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type StreamkapAPI interface {
	GetAccessToken(clientID, secret string) (*Token, error)
	SetToken(token *Token)

	//Source APIs
	CreateSource(ctx context.Context, reqPayload Source) (*Source, error)
	UpdateSource(ctx context.Context, sourceID string, reqPayload Source) (*Source, error)
	GetSource(ctx context.Context, sourceID string) (*Source, error)
	ListSources(ctx context.Context) ([]Source, error)
	DeleteSource(ctx context.Context, sourceID string) error

	// Destination APIs
	CreateDestination(ctx context.Context, reqPayload Destination) (*Destination, error)
	UpdateDestination(ctx context.Context, destinationID string, reqPayload Destination) (*Destination, error)
	GetDestination(ctx context.Context, destinationID string) (*Destination, error)
	ListDestinations(ctx context.Context) ([]Destination, error)
	DeleteDestination(ctx context.Context, destinationID string) error

	// Pipeline APIs
	CreatePipeline(ctx context.Context, reqPayload Pipeline) (*Pipeline, error)
	UpdatePipeline(ctx context.Context, pipelineID string, reqPayload Pipeline) (*Pipeline, error)
	GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error)
	ListPipelines(ctx context.Context) ([]Pipeline, error)
	DeletePipeline(ctx context.Context, pipelineID string) error

	// Transform APIs
	CreateTransform(ctx context.Context, reqPayload CreateTransformRequest) (*Transform, error)
	UpdateTransform(ctx context.Context, transformID string, reqPayload UpdateTransformRequest) (*Transform, error)
	GetTransform(ctx context.Context, transformID string) (*Transform, error)
	ListTransforms(ctx context.Context) ([]Transform, error)
	DeleteTransform(ctx context.Context, transformID string) error
	GetTransformImplementationDetails(ctx context.Context, transformID string) (*TransformImplementationDetailsResponse, error)
	UpdateTransformImplementationDetails(ctx context.Context, transformID string, details TransformImplementationDetails) (*TransformImplementationDetails, error)
	DeployTransformPreview(ctx context.Context, transformID string, versionID string, replayWindow string) error
	DeployTransformLive(ctx context.Context, transformID string, versionID string) error
	GetTransformJobStatus(ctx context.Context, transformID string) (*TransformJobStatus, error)

	// Tags APIs
	GetTag(ctx context.Context, TagID string) (*Tag, error)
	ListTags(ctx context.Context, filters TagListFilters) ([]Tag, error)
	CreateTag(ctx context.Context, reqPayload Tag) (*Tag, error)
	UpdateTag(ctx context.Context, tagID string, reqPayload Tag) (*Tag, error)
	DeleteTag(ctx context.Context, tagID string) error

	// Topic APIs
	GetTopic(ctx context.Context, TopicID string) (*Topic, error)
	GetTopicDetailed(ctx context.Context, TopicID string) (*TopicDetailed, error)
	UpdateTopic(ctx context.Context, TopicID string, reqPayload Topic) (*Topic, error)
	DeleteTopic(ctx context.Context, TopicID string) error
	ListTopics(ctx context.Context, params *TopicListParams) (*TopicDetailsResponse, error)
	GetTopicTableMetrics(ctx context.Context, req TopicTableMetricsRequest) (TopicTableMetricsResponse, error)

	// Kafka User APIs
	CreateKafkaUser(ctx context.Context, reqPayload CreateKafkaUserRequest) (*KafkaUser, error)
	GetKafkaUser(ctx context.Context, username string) (*KafkaUser, error)
	ListKafkaUsers(ctx context.Context) ([]KafkaUser, error)
	UpdateKafkaUser(ctx context.Context, username string, reqPayload UpdateKafkaUserRequest) (*KafkaUser, error)
	DeleteKafkaUser(ctx context.Context, username string) error

	// Client Credential APIs
	CreateClientCredential(ctx context.Context, reqPayload CreateClientCredentialRequest) (*ClientCredential, error)
	GetClientCredential(ctx context.Context, clientID string) (*ClientCredential, error)
	ListClientCredentials(ctx context.Context) ([]ClientCredential, error)
	DeleteClientCredential(ctx context.Context, clientID string) error

	// Role APIs
	ListRoles(ctx context.Context) ([]Role, error)
}

type APIErrorResponse struct {
	Detail string `json:"detail"`
}

// APIError is a non-2xx response from the Streamkap API. The HTTP status is
// carried next to the body detail instead of being flattened into the message,
// so callers classify on the status (retryable, gone, unauthorized) rather than
// substring-matching prose: matching "429"/"502" against the text retried any
// 400 whose detail happened to contain those digits (e.g. "port 4290 invalid")
// and never retried a 429 whose detail omitted them.
type APIError struct {
	StatusCode int
	Detail     string
	RequestID  string
}

func (e *APIError) Error() string {
	detail := e.Detail
	if detail == "" {
		detail = http.StatusText(e.StatusCode)
	}
	if e.RequestID != "" {
		return fmt.Sprintf("%s (HTTP %d, request_id=%s)", detail, e.StatusCode, e.RequestID)
	}
	return fmt.Sprintf("%s (HTTP %d)", detail, e.StatusCode)
}

// notFoundDetails are the phrasings the backend uses for a record that is gone
// when it answers with a status other than 404 (it is not consistent across
// endpoints).
var notFoundDetails = []string{
	"not found",
	"does not exist",
	"no such",
}

// isAlreadyExists reports whether Create failed because the backend already
// holds a record with this name.
func isAlreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

// adoptRefusedError is what every Create returns on an "already exists"
// collision. No resource auto-adopts the existing record, and the reason is
// worth stating once.
//
// Streamkap enforces unique names per tenant (per tenant/service for pipelines
// and transforms), so the collision means one of two things:
//
//  1. A previous apply created the record but lost the response (network,
//     timeout). State believes the resource is absent; the backend has it.
//     Recovery is `terraform import`.
//
//  2. A `lifecycle { create_before_destroy = true }` replace is in flight:
//     Terraform deposed the old live instance (id=X) and asked us to create the
//     new one, which collides on name with the still-present backend record X.
//
// Adopting — looking the record up by name and returning it as the create
// result — is right for (1) and destructive in (2): the new live state entry
// inherits the deposed entry's backend id, so the next step of the apply,
// destroying the deposed entry, deletes the record the live state now points at.
// That is not hypothetical. A customer trace showed four deposed pipeline
// entries all pointing at a single backend pipeline after repeated adoption.
//
// The two cases are indistinguishable from inside the API client, which has no
// state visibility, and a content-comparison adopt collapses to the same data
// loss under taint + create_before_destroy. So we fail loudly and hand the user
// the recovery steps: loud failure beats silent data loss.
//
// adoptConflict describes the resource that collided, so the shared message can
// name it precisely.
type adoptConflict struct {
	Kind        string // noun used in the message: "source", "pipeline", …
	Name        string
	UniqueScope string // what the name must be unique within: "tenant", "tenant/service"
	ImportAddr  string // address + id shape for `terraform import`
	ImportNote  string // optional extra guidance appended to the import bullet
	ListPath    string // endpoint for looking the id up
}

func adoptRefusedError(c adoptConflict, err error) error {
	importNote := c.ImportNote
	if importNote != "" {
		importNote = " " + importNote
	}
	return fmt.Errorf(
		"streamkap %[1]s %[2]q already exists on the backend, and auto-adoption is unsafe (it would risk destroying the live %[1]s under create_before_destroy). Recovery options:\n"+
			"  • If this is a `lifecycle { create_before_destroy = true }` replace: remove that directive — Streamkap enforces unique %[1]s names per %[3]s, so a new and an old %[1]s cannot coexist by name. Use the default destroy-then-create, or rename so the two can briefly coexist.\n"+
			"  • If you have deposed entries from earlier failed applies (shown in `terraform plan` as `<address> (destroy deposed <key>)`), they point at the same backend record and must leave state before any retry can succeed: `terraform state rm '<resource_address>'`. See docs/MIGRATION.md → \"Known limitations\".\n"+
			"  • If this is recovering from an apply that lost its response: run `terraform import %[4]s`.%[5]s Find the id in the Streamkap UI or via `GET %[6]s?partial_name=%[2]s`, then re-run apply.\n"+
			"Original backend error: %[7]w",
		c.Kind, c.Name, c.UniqueScope, c.ImportAddr, importNote, c.ListPath, err,
	)
}

// IsNotFound reports whether err says the addressed record does not exist.
// Delete paths treat it as success: the desired end state (absent) already
// holds, and failing instead forces the user into `terraform state rm` whenever
// a resource was removed out of band.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusNotFound {
		return true
	}
	// Only trust the wording on client errors. A 5xx mentioning "not found"
	// is a backend failure, not a statement about the record.
	if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
		return false
	}
	detail := strings.ToLower(apiErr.Detail)
	for _, phrase := range notFoundDetails {
		if strings.Contains(detail, phrase) {
			return true
		}
	}
	return false
}

type Config struct {
	BaseURL        string `mapstructure:"base_url"`
	AdminTenantID  string
	AdminServiceID string
}

// tokenRenewSkew renews the access token this long before it actually expires,
// so a request issued just under the wire doesn't cross the boundary in flight.
const tokenRenewSkew = 60 * time.Second

type streamkapAPI struct {
	cfg    *Config
	client *http.Client

	// mu guards the token and the credentials used to renew it. The client is
	// shared by every resource and Terraform runs CRUD concurrently, so the
	// per-request read and the renewal write must be serialized.
	mu          sync.Mutex
	token       *Token
	tokenExpiry time.Time
	clientID    string
	secret      string

	// renewMu single-flights renewal: on a token expiry every in-flight
	// request 401s at once, and without this they would all hit the auth
	// endpoint simultaneously.
	renewMu sync.Mutex
}

func NewClient(cfg *Config) StreamkapAPI {
	return &streamkapAPI{
		cfg:    cfg,
		client: http.DefaultClient,
	}
}

func (s *streamkapAPI) SetToken(token *Token) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.token = token
	s.tokenExpiry = time.Time{}
	// Frontegg reports the lifetime in seconds; `expires` is a formatted string
	// whose layout we don't pin. With no lifetime we simply never renew ahead of
	// time and fall back to renewing when a 401 arrives.
	if token != nil && token.ExpiresIn > 0 {
		s.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
}

// bearer returns the access token currently on the client and whether it is
// close enough to expiry to renew before using it.
func (s *streamkapAPI) bearer() (token string, expiring bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token == nil {
		return "", false
	}
	if !s.tokenExpiry.IsZero() && time.Now().After(s.tokenExpiry.Add(-tokenRenewSkew)) {
		return s.token.AccessToken, true
	}
	return s.token.AccessToken, false
}

// credentials returns the client credentials captured by GetAccessToken. They
// are empty when the caller only ever handed us a token via SetToken, in which
// case the client cannot renew and a 401 is returned to the caller as-is.
func (s *streamkapAPI) credentials() (clientID, secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientID, s.secret
}

func (s *streamkapAPI) canRenewToken() bool {
	clientID, secret := s.credentials()
	return clientID != "" && secret != ""
}

// renewToken exchanges the stored client credentials for a fresh access token.
// staleToken is the bearer whose request just failed: if another goroutine
// renewed past it while this one waited on renewMu, that renewal serves us too
// and we return without a second call to the auth endpoint.
func (s *streamkapAPI) renewToken(ctx context.Context, staleToken string) error {
	s.renewMu.Lock()
	defer s.renewMu.Unlock()

	if current, _ := s.bearer(); current != "" && current != staleToken {
		return nil
	}

	clientID, secret := s.credentials()
	if clientID == "" || secret == "" {
		return errors.New("renewToken: the API client holds no client credentials, cannot renew the access token")
	}

	token, err := s.authenticate(ctx, clientID, secret)
	if err != nil {
		return fmt.Errorf("renewToken: re-authentication failed: %w", err)
	}
	s.SetToken(token)
	tflog.Debug(ctx, "Renewed the Streamkap access token")
	return nil
}

func (s *streamkapAPI) doRequest(ctx context.Context, req *http.Request, result any) error {
	return s.do(ctx, req, result, true)
}

// do sends req and decodes the response. When renewal is allowed (every call
// but the token exchange itself, which would recurse), a token that is about to
// expire is renewed up front, and a 401 triggers one renewal plus a replay of
// the original request: an apply lasting longer than the token TTL used to fail
// every remaining resource with an opaque 401.
func (s *streamkapAPI) do(ctx context.Context, req *http.Request, result any, allowRenew bool) error {
	bearer, expiring := s.bearer()
	if allowRenew && expiring && s.canRenewToken() {
		if err := s.renewToken(ctx, bearer); err != nil {
			// The current token may still be inside the skew window and work.
			// Don't fail a request we haven't even sent — if the token really is
			// dead the 401 path below renews again and surfaces the failure.
			tflog.Warn(ctx, fmt.Sprintf("Proactive token renewal failed, continuing with the current token: %s", err))
		} else {
			bearer, _ = s.bearer()
		}
	}

	err := s.send(ctx, req, result, bearer)
	if !allowRenew || !isUnauthorized(err) || !s.canRenewToken() {
		return err
	}

	if renewErr := s.renewToken(ctx, bearer); renewErr != nil {
		return fmt.Errorf("%w (renewing the access token after the 401 also failed: %v)", err, renewErr)
	}
	if rewindErr := rewindBody(req); rewindErr != nil {
		return fmt.Errorf("%w (could not replay the request after renewing the access token: %v)", err, rewindErr)
	}
	fresh, _ := s.bearer()
	return s.send(ctx, req, result, fresh)
}

func (s *streamkapAPI) send(ctx context.Context, req *http.Request, result any, bearer string) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if bearer != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearer))
	}

	if s.cfg.AdminTenantID != "" {
		req.Header.Set("X-Admin-Tenant-Id", s.cfg.AdminTenantID)
	}
	if s.cfg.AdminServiceID != "" {
		req.Header.Set("X-Admin-Service-Id", s.cfg.AdminServiceID)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if requestID != "" {
		tflog.Debug(ctx, fmt.Sprintf("%s %s → %d (request_id=%s)", req.Method, req.URL, resp.StatusCode, requestID))
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("%s %s → %d: failed to read response body: %w", req.Method, req.URL, resp.StatusCode, readErr)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// The backend's normal error shape is `{"detail": "..."}`, but a non-2xx
		// can also come from an intermediate layer (nginx 502/504, load balancer,
		// auth redirect) and arrive as HTML. Returning only the bare JSON parse
		// error ("invalid character '<' looking for beginning of value") strips
		// every actionable hint, which has been a recurring debugging dead-end.
		var apiErr APIErrorResponse
		if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil && apiErr.Detail != "" {
			return &APIError{StatusCode: resp.StatusCode, Detail: apiErr.Detail, RequestID: requestID}
		}
		tflog.Debug(ctx,
			fmt.Sprintf("%s %s → %d: response body not JSON: %s",
				req.Method, req.URL, resp.StatusCode, snippet(body)),
		)
		return &APIError{
			StatusCode: resp.StatusCode,
			Detail:     fmt.Sprintf("%s %s: non-JSON response body: %s", req.Method, req.URL, snippet(body)),
			RequestID:  requestID,
		}
	}

	// A 2xx with an empty body is a success with nothing to decode — the
	// kafka-user and client-credential deletes answer exactly that. Handing it
	// to the decoder reports io.EOF and turns a successful call into a failure.
	if len(bytes.TrimSpace(body)) == 0 {
		tflog.Debug(ctx, fmt.Sprintf("%s %s → %d with an empty body; nothing to decode", req.Method, req.URL, resp.StatusCode))
		return nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("%s %s → %d: failed to decode response body: %w", req.Method, req.URL, resp.StatusCode, err)
	}
	return nil
}

func isUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// rewindBody restores req.Body so the request can be sent again (retry, or
// replay after a token renewal). http.NewRequest* populates GetBody for the
// in-memory body types this client uses, so the payload is always reproducible;
// a request built any other way fails loudly here rather than silently
// replaying an empty body.
func rewindBody(req *http.Request) error {
	if req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	if req.GetBody == nil {
		return fmt.Errorf("rewindBody: %s %s carries a body that cannot be replayed", req.Method, req.URL)
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("rewindBody: failed to rewind the body of %s %s: %w", req.Method, req.URL, err)
	}
	req.Body = body
	return nil
}

// snippet trims a response body to a single-line preview suitable for error
// messages. Used for surfacing HTML/non-JSON error bodies that would
// otherwise be lost behind a bare json-decoder error.
func snippet(b []byte) string {
	const maxLen = 200
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > maxLen {
		s = s[:maxLen] + "...[truncated]"
	}
	if s == "" {
		s = "(empty body)"
	}
	return s
}

// doRequestWithRetry wraps doRequest with retry logic for transient errors.
// Use this for Create/Update/Delete operations only, not for Read.
func (s *streamkapAPI) doRequestWithRetry(ctx context.Context, req *http.Request, result any) error {
	cfg := DefaultRetryConfig()

	return RetryWithBackoff(ctx, cfg, func() error {
		if err := rewindBody(req); err != nil {
			return err
		}
		return s.doRequest(ctx, req, result)
	})
}

// deleteResource issues a DELETE and treats "the record is already gone" as
// success. Terraform's desired end state after a destroy is absence; failing
// because someone deleted the resource in the UI first only forces the user to
// run `terraform state rm`. opName names the calling API method so the debug
// log and any error identify it.
func (s *streamkapAPI) deleteResource(ctx context.Context, opName, reqURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("%s: failed to build the delete request: %w", opName, err)
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"%s request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		opName,
		req.Method,
		req.URL.String(),
	))

	var resp json.RawMessage
	if err := s.doRequestWithRetry(ctx, req, &resp); err != nil {
		if IsNotFound(err) {
			tflog.Info(ctx, fmt.Sprintf("%s: the backend reports the record does not exist; treating the delete as done", opName))
			return nil
		}
		return err
	}
	return nil
}
