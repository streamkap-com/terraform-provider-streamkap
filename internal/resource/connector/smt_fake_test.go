package connector_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

// fixtureConfig is the released PostgreSQL config with the chain built from
// the contract corpus instead of the pinned catalog, whose types are all
// publishable: false. Everything else, the generated model included, is the
// real resource.
type fixtureConfig struct {
	*source.PostgreSQLConfig
	chain *smt.ChainSchema
}

func (c *fixtureConfig) GetSMTChain() *smt.ChainSchema { return c.chain }

// destinationFixtureConfig is the same for the released Snowflake destination.
type destinationFixtureConfig struct {
	*destination.SnowflakeConfig
	chain *smt.ChainSchema
}

func (c *destinationFixtureConfig) GetSMTChain() *smt.ChainSchema { return c.chain }

func fixtureCatalog(t *testing.T) smt.Catalog {
	t.Helper()
	corpus, err := smt.LoadCorpus(filepath.Join("..", "..", "smt", "testdata", "contract_corpus.json"))
	require.NoError(t, err)
	cat := smt.NewCatalog(corpus)
	require.Len(t, cat, 2)
	return cat
}

// fakeAPI is an in-memory Streamkap API for the connector endpoints the base
// resource calls plus the chain surface. It echoes config the way a JSON API
// would (numbers as float64), guards chain writes by revision, records every
// secret operation and counts writes so a test can prove a rejected plan
// never reached it.
type fakeAPI struct {
	api.StreamkapAPI
	mu      sync.Mutex
	cat     smt.Catalog
	next    int
	sources map[string]*api.Source
	kinds   map[string]string
	// flatKeysSeen records, per source id, the transforms.* and predicates.*
	// keys the last create or update carried.
	flatKeysSeen map[string][]string
	chains       map[string]*api.SMTChainSnapshot
	revisions    map[string]int
	secrets      map[string]map[string]string
	ops          []recordedOp
	writes       int
	chainReads   int
	// projectFlat makes reads echo the chain into a legacy flat key, the way a
	// backend adapter might; the provider must never let that become state.
	projectFlat bool
	// failNextChainWrite makes the next PutSMTChain fail after the connector
	// itself was created.
	failNextChainWrite bool
	// bumpRevisionBeforeNextWrite simulates an edit made elsewhere between the
	// plan's refresh and the apply: the next PutSMTChain sees a newer revision.
	bumpRevisionBeforeNextWrite bool
	// nextWriteIssues makes the next PutSMTChain fail validation with them.
	nextWriteIssues []api.SMTValidationIssue
}

func newFakeAPI(cat smt.Catalog) *fakeAPI {
	return &fakeAPI{
		cat:          cat,
		sources:      map[string]*api.Source{},
		kinds:        map[string]string{},
		flatKeysSeen: map[string][]string{},
		chains:       map[string]*api.SMTChainSnapshot{},
		revisions:    map[string]int{},
		secrets:      map[string]map[string]string{},
	}
}

func (f *fakeAPI) id(prefix string) string {
	f.next++
	return fmt.Sprintf("%s-%d", prefix, f.next)
}

func roundTrip[T any](in T) T {
	raw, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func flatKeys(cfg map[string]any) []string {
	var keys []string
	for k := range cfg {
		if strings.HasPrefix(k, "transforms.") || strings.HasPrefix(k, "predicates.") {
			keys = append(keys, k)
		}
	}
	return keys
}

func (f *fakeAPI) CreateSource(_ context.Context, req api.Source) (*api.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	stored := roundTrip(req)
	stored.ID = f.id("conn")
	stored.ConnectorStatus = "Active"
	f.sources[stored.ID] = &stored
	f.kinds[stored.ID] = "sources"
	f.flatKeysSeen[stored.ID] = flatKeys(req.Config)
	out := f.view(stored.ID)
	return &out, nil
}

// Destinations share the source store; only the kind the chain endpoints
// are addressed under differs.
func (f *fakeAPI) CreateDestination(ctx context.Context, req api.Destination) (*api.Destination, error) {
	src, err := f.CreateSource(ctx, api.Source(req))
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.kinds[src.ID] = "destinations"
	f.mu.Unlock()
	out := api.Destination(*src)
	return &out, nil
}

func (f *fakeAPI) GetDestination(ctx context.Context, id string) (*api.Destination, error) {
	src, err := f.GetSource(ctx, id)
	if err != nil || src == nil {
		return nil, err
	}
	out := api.Destination(*src)
	return &out, nil
}

func (f *fakeAPI) UpdateDestination(ctx context.Context, id string, req api.Destination) (*api.Destination, error) {
	src, err := f.UpdateSource(ctx, id, api.Source(req))
	if err != nil {
		return nil, err
	}
	out := api.Destination(*src)
	return &out, nil
}

func (f *fakeAPI) DeleteDestination(ctx context.Context, id string) error {
	return f.DeleteSource(ctx, id)
}

func (f *fakeAPI) GetSource(_ context.Context, id string) (*api.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sources[id]; !ok {
		return nil, nil
	}
	out := f.view(id)
	return &out, nil
}

func (f *fakeAPI) UpdateSource(_ context.Context, id string, req api.Source) (*api.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	prior, ok := f.sources[id]
	if !ok {
		return nil, &api.APIError{StatusCode: http.StatusNotFound, Detail: "source not found"}
	}
	stored := roundTrip(req)
	stored.ID = id
	stored.ConnectorStatus = prior.ConnectorStatus
	f.sources[id] = &stored
	f.flatKeysSeen[id] = flatKeys(req.Config)
	out := f.view(id)
	return &out, nil
}

func (f *fakeAPI) DeleteSource(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	delete(f.sources, id)
	delete(f.chains, id)
	return nil
}

// recordedOp is one secret operation the fake applied, by server id.
type recordedOp struct {
	InstanceID string
	Pointer    string
	Clear      bool
	Value      string
}

// view is the read projection of a source. With projectFlat set and a chain
// present, the legacy flat key carries a value the provider did not send.
func (f *fakeAPI) view(id string) api.Source {
	out := roundTrip(*f.sources[id])
	if f.projectFlat && f.chains[id] != nil && len(f.chains[id].Transforms) > 0 {
		out.Config["transforms.ValueToKey.fields.include.list"] = "projected-from-chain"
	}
	return out
}

func (f *fakeAPI) GetSMTChain(_ context.Context, kind, id string) (*api.SMTChainRead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chainReads++
	if kind != f.kinds[id] {
		return nil, fmt.Errorf("chain read under kind %q for a %s connector", kind, f.kinds[id])
	}
	if _, ok := f.sources[id]; !ok {
		return nil, &api.APIError{StatusCode: http.StatusNotFound, Detail: "source not found"}
	}
	if f.chains[id] == nil {
		return nil, smtError(http.StatusNotFound, "smt_chain_absent", "The destination has no chain")
	}
	return f.readOf(id), nil
}

// chainOf is the stored desired snapshot; a connector without one has an
// empty chain at revision 0.
func (f *fakeAPI) chainOf(id string) *api.SMTChainSnapshot {
	if f.chains[id] == nil {
		f.chains[id] = &api.SMTChainSnapshot{ChainRevision: strconv.Itoa(f.revisions[id]), Transforms: []api.SMTInstanceRead{}}
	}
	return f.chains[id]
}

func (f *fakeAPI) readOf(id string) *api.SMTChainRead {
	snapshot := roundTrip(*f.chainOf(id))
	return &api.SMTChainRead{Desired: snapshot, ApplicationStatus: "pending"}
}

// smtError is the structured envelope the chain routes answer with.
func smtError(status int, code, message string, issues ...api.SMTValidationIssue) error {
	detail, err := json.Marshal(api.SMTErrorEnvelope{Code: code, Message: message, Issues: issues})
	if err != nil {
		panic(err)
	}
	return &api.APIError{StatusCode: status, Detail: message, DetailJSON: detail}
}

// PutSMTChain replaces the chain under the revision guard and applies the
// secret operations each instance carries, checking every pointer against
// the persisted config the way the backend's SmtSecretPointer validation
// would.
func (f *fakeAPI) PutSMTChain(_ context.Context, kind, id string, w api.SMTChainWrite) (*api.SMTChainRead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if kind != f.kinds[id] {
		return nil, fmt.Errorf("chain write under kind %q for a %s connector", kind, f.kinds[id])
	}
	if _, ok := f.sources[id]; !ok {
		return nil, &api.APIError{StatusCode: http.StatusNotFound, Detail: "source not found"}
	}
	if f.failNextChainWrite {
		f.failNextChainWrite = false
		return nil, &api.APIError{StatusCode: http.StatusBadGateway, Detail: "chain service unavailable"}
	}
	if len(f.nextWriteIssues) > 0 {
		issues := f.nextWriteIssues
		f.nextWriteIssues = nil
		return nil, smtError(http.StatusUnprocessableEntity, "smt_validation_failed", "The chain was not saved; fix the issues and send it again", issues...)
	}
	exists := f.chains[id] != nil
	current := f.chainOf(id)
	if f.bumpRevisionBeforeNextWrite {
		f.bumpRevisionBeforeNextWrite = false
		f.revisions[id]++
		current.ChainRevision = strconv.Itoa(f.revisions[id])
	}
	if w.ExpectedRevision == nil && exists {
		return nil, smtError(http.StatusConflict, "smt_expected_revision_required", "The chain exists; send its revision as expected_revision")
	}
	if w.ExpectedRevision != nil && !exists {
		return nil, smtError(http.StatusNotFound, "smt_chain_absent", "The destination has no chain")
	}
	if w.ExpectedRevision != nil && *w.ExpectedRevision != current.ChainRevision {
		return nil, smtError(http.StatusConflict, "smt_revision_conflict", "The chain changed since it was read; read it again before saving",
			api.SMTValidationIssue{Code: "smt_revision_conflict", Message: "expected revision is stale", Pointer: "/expected_revision"})
	}
	instances, err := f.persist(current.Transforms, w.Transforms)
	if err != nil {
		return nil, err
	}
	// Operations apply against the persisted config of the instance that
	// carried them.
	for i, inst := range instances {
		for j, op := range w.Transforms[i].SecretOperations {
			addr, err := smt.ResolvePointer(f.cat[inst.Type], op.Pointer)
			if err != nil || !smt.RowExists(f.cat[inst.Type], addr, inst.Config) {
				return nil, smtError(http.StatusUnprocessableEntity, "smt_validation_failed", "The chain was not saved; fix the issues and send it again",
					api.SMTValidationIssue{Code: "smt_unknown_secret_row", Message: "Pointer addresses a row key that is not present", Pointer: fmt.Sprintf("/transforms/%d/secret_operations/%d/pointer", i, j)})
			}
			if f.secrets[inst.ID] == nil {
				f.secrets[inst.ID] = map[string]string{}
			}
			rec := recordedOp{InstanceID: inst.ID, Pointer: op.Pointer}
			if op.Operation == api.SMTSecretClear {
				delete(f.secrets[inst.ID], op.Pointer)
				rec.Clear = true
			} else {
				f.secrets[inst.ID][op.Pointer] = op.Value
				rec.Value = op.Value
			}
			f.ops = append(f.ops, rec)
		}
	}
	f.revisions[id]++
	f.chains[id] = &api.SMTChainSnapshot{ChainRevision: strconv.Itoa(f.revisions[id]), Transforms: instances}
	return f.readOf(id), nil
}

// persist applies a chain write: an echoed id keeps its identity, a missing
// id mints one, and an instance the write omits is gone.
func (f *fakeAPI) persist(prior []api.SMTInstanceRead, chain []api.SMTInstanceWrite) ([]api.SMTInstanceRead, error) {
	byID := map[string]api.SMTInstanceRead{}
	for _, p := range prior {
		byID[p.ID] = p
	}
	out := make([]api.SMTInstanceRead, 0, len(chain))
	for _, w := range chain {
		spec, ok := f.cat[w.Type]
		if !ok {
			return nil, fmt.Errorf("unknown type %s", w.Type)
		}
		if w.Name == "" {
			return nil, smtError(http.StatusUnprocessableEntity, "smt_validation_failed", "The chain was not saved; fix the issues and send it again",
				api.SMTValidationIssue{Code: "smt_config_missing", Message: "Field required", Pointer: fmt.Sprintf("/transforms/%d/name", len(out))})
		}
		read := api.SMTInstanceRead{Kind: "typed", Type: w.Type, SchemaVersion: spec.SchemaVersion, CodecVersion: spec.CodecVersion, Name: w.Name, Enabled: w.Enabled, Config: roundTrip(w.Config), ConfiguredSecretPaths: []string{}, Predicate: json.RawMessage("null")}
		if w.ID != "" {
			old, ok := byID[w.ID]
			if !ok {
				return nil, fmt.Errorf("instance %s not found", w.ID)
			}
			if old.Type != w.Type {
				return nil, fmt.Errorf("instance %s cannot change type", w.ID)
			}
			read.ID, read.Alias = old.ID, old.Alias
		} else {
			read.ID = f.id("inst")
			read.Alias = fmt.Sprintf("%s_%s", w.Type, read.ID)
		}
		out = append(out, read)
	}
	return out, nil
}

// seedChain creates a connector with a chain outside Terraform, the way the
// UI would.
func (f *fakeAPI) seedChain(t *testing.T, chain []api.SMTInstanceWrite) string {
	t.Helper()
	ctx := context.Background()
	src, err := f.CreateSource(ctx, api.Source{Name: "seeded", Connector: "postgresql", Config: map[string]any{
		"database.hostname.user.defined": "db.invalid", "database.user.user.defined": "u", "database.password.user.defined": "p",
		"database.dbname.user.defined": "d", "schema.include.list.user.defined": "public", "table.include.list.user.defined": "public.t",
	}})
	require.NoError(t, err)
	_, err = f.PutSMTChain(ctx, "sources", src.ID, api.SMTChainWrite{Transforms: chain})
	require.NoError(t, err)
	return src.ID
}

// setPredicate attaches a predicate to an instance outside Terraform.
func (f *fakeAPI) setPredicate(id, instanceID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.chainOf(id).Transforms {
		if f.chains[id].Transforms[i].ID == instanceID {
			f.chains[id].Transforms[i].Predicate = json.RawMessage(`{"id":"pred-1","alias":"P1","type":"contract_fixture_nested","schema_version":1,"codec_version":1,"config":{},"configured_secret_paths":[],"negate":false}`)
		}
	}
}

// rejectNextWrite makes the next PutSMTChain answer 422 with the given
// issues.
func (f *fakeAPI) rejectNextWrite(issues ...api.SMTValidationIssue) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextWriteIssues = issues
}

// applySecret rotates one secret outside Terraform.
func (f *fakeAPI) applySecret(t *testing.T, id, instanceID, pointer, value string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.secrets[instanceID] == nil {
		f.secrets[instanceID] = map[string]string{}
	}
	f.secrets[instanceID][pointer] = value
	f.ops = append(f.ops, recordedOp{InstanceID: instanceID, Pointer: pointer, Value: value})
	_ = id
}

func (f *fakeAPI) opsSince(n int) []recordedOp {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedOp(nil), f.ops[n:]...)
}

func (f *fakeAPI) secretValue(instanceID, pointer string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.secrets[instanceID][pointer]
}

func (f *fakeAPI) flatKeysSent(id string) map[string]bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]bool{}
	for _, k := range f.flatKeysSeen[id] {
		out[k] = true
	}
	return out
}

func (f *fakeAPI) chainReadCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.chainReads
}

// replaceConfig sets a source's stored config outside Terraform.
func (f *fakeAPI) replaceConfig(id string, cfg map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sources[id].Config = roundTrip(cfg)
}

// setFlat edits a flat config key on every stored source, outside Terraform.
func (f *fakeAPI) setFlat(key string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, src := range f.sources {
		src.Config[key] = value
	}
}

func (f *fakeAPI) sourceIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for id := range f.sources {
		ids = append(ids, id)
	}
	return ids
}

func (f *fakeAPI) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes
}

func (f *fakeAPI) instances(id string) []api.SMTInstanceRead {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.chains[id] == nil {
		return nil
	}
	return append([]api.SMTInstanceRead(nil), f.chains[id].Transforms...)
}

// testProvider registers the real connector resources over the fake client.
type testProvider struct {
	client  *fakeAPI
	configs []connector.ConnectorConfig
}

func (p *testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "streamkap"
}

func (p *testProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p *testProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
}

func (p *testProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

func (p *testProvider) Resources(context.Context) []func() resource.Resource {
	var out []func() resource.Resource
	for _, cfg := range p.configs {
		out = append(out, func() resource.Resource { return connector.NewBaseConnectorResource(cfg) })
	}
	return out
}

func fixtureResource(t *testing.T) resource.Resource {
	t.Helper()
	chain, err := smt.BuildChain(fixtureCatalog(t))
	require.NoError(t, err)
	return connector.NewBaseConnectorResource(&fixtureConfig{PostgreSQLConfig: &source.PostgreSQLConfig{}, chain: chain})
}

func fixtureMappings(t *testing.T) map[string]string {
	t.Helper()
	return (&source.PostgreSQLConfig{}).GetFieldMappings()
}

// newFixture builds the integrated resource over the corpus catalog and the
// fake client.
func newFixture(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeAPI) {
	t.Helper()
	cat := fixtureCatalog(t)
	chain, err := smt.BuildChain(cat)
	require.NoError(t, err)
	fake := newFakeAPI(cat)
	p := &testProvider{client: fake, configs: []connector.ConnectorConfig{
		&fixtureConfig{PostgreSQLConfig: &source.PostgreSQLConfig{}, chain: chain},
		&destinationFixtureConfig{SnowflakeConfig: &destination.SnowflakeConfig{}, chain: chain},
	}}
	return map[string]func() (tfprotov6.ProviderServer, error){
		"streamkap": providerserver.NewProtocol6WithError(p),
	}, fake
}
