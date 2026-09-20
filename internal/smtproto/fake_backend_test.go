package smtproto

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// fakeBackend is an in-memory server. It stores what the wire projection
// sends, echoes it back the way a JSON API would (numbers as float64), and
// records every secret operation so a test can count them.
type fakeBackend struct {
	mu         sync.Mutex
	cat        Catalog
	next       int
	connectors map[string][]InstanceRead
	secrets    map[string]map[string]string
	ops        []ResolvedSecretOp
	writes     int
}

func newFakeBackend(cat Catalog) *fakeBackend {
	return &fakeBackend{cat: cat, connectors: map[string][]InstanceRead{}, secrets: map[string]map[string]string{}}
}

func (f *fakeBackend) id(prefix string) string {
	f.next++
	return fmt.Sprintf("%s-%d", prefix, f.next)
}

func (f *fakeBackend) Create(_ context.Context, chain []InstanceWrite) (string, []InstanceRead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	id := f.id("conn")
	reads, err := f.persist(nil, chain)
	if err != nil {
		return "", nil, err
	}
	f.connectors[id] = reads
	return id, reads, nil
}

func (f *fakeBackend) Read(_ context.Context, id string) ([]InstanceRead, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	reads, ok := f.connectors[id]
	return reads, ok, nil
}

func (f *fakeBackend) Update(_ context.Context, id string, chain []InstanceWrite) ([]InstanceRead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	prior, ok := f.connectors[id]
	if !ok {
		return nil, fmt.Errorf("connector %s not found", id)
	}
	reads, err := f.persist(prior, chain)
	if err != nil {
		return nil, err
	}
	f.connectors[id] = reads
	return reads, nil
}

func (f *fakeBackend) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	delete(f.connectors, id)
	return nil
}

// ApplySecrets checks each pointer against the persisted config the way the
// backend's SmtSecretPointer validation would, then records it.
func (f *fakeBackend) ApplySecrets(_ context.Context, id string, ops []ResolvedSecretOp) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	for _, op := range ops {
		var inst *InstanceRead
		for i := range f.connectors[id] {
			if f.connectors[id][i].ID == op.InstanceID {
				inst = &f.connectors[id][i]
			}
		}
		if inst == nil {
			return fmt.Errorf("instance %s not on connector %s", op.InstanceID, id)
		}
		addr, err := ResolvePointer(f.cat[inst.Type], op.Pointer)
		if err != nil {
			return err
		}
		if !RowExists(f.cat[inst.Type], addr, inst.Config) {
			return fmt.Errorf("pointer %s names no row on %s", op.Pointer, op.InstanceID)
		}
		if f.secrets[op.InstanceID] == nil {
			f.secrets[op.InstanceID] = map[string]string{}
		}
		if op.Clear {
			delete(f.secrets[op.InstanceID], op.Pointer)
		} else {
			f.secrets[op.InstanceID][op.Pointer] = op.Value
		}
		f.ops = append(f.ops, op)
	}
	return nil
}

// persist applies a chain write: an echoed id keeps its identity, a missing
// id mints one, and an instance the write omits is gone.
func (f *fakeBackend) persist(prior []InstanceRead, chain []InstanceWrite) ([]InstanceRead, error) {
	byID := map[string]InstanceRead{}
	for _, p := range prior {
		byID[p.ID] = p
	}
	out := make([]InstanceRead, 0, len(chain))
	for _, w := range chain {
		spec, ok := f.cat[w.Type]
		if !ok {
			return nil, fmt.Errorf("unknown type %s", w.Type)
		}
		read := InstanceRead{Type: w.Type, SchemaVersion: spec.SchemaVersion, Name: w.Name}
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
		raw, err := json.Marshal(w.Config)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &read.Config); err != nil {
			return nil, err
		}
		out = append(out, read)
	}
	return out, nil
}

// opsSince returns the operations recorded after the first n.
func (f *fakeBackend) opsSince(n int) []ResolvedSecretOp {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ResolvedSecretOp(nil), f.ops[n:]...)
}

// testProvider registers only the prototype resource, with no configuration
// and no API client, so terraform can drive it against the fake.
type testProvider struct {
	res *Resource
}

func (p *testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "streamkap"
}

func (p *testProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p *testProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (p *testProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

func (p *testProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return p.res }}
}

func providerFactories(res *Resource) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"streamkap": providerserver.NewProtocol6WithError(&testProvider{res: res}),
	}
}
