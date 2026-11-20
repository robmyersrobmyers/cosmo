package module_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wundergraph/cosmo/router-tests/testenv"
	"github.com/wundergraph/cosmo/router/core"
	nodev1 "github.com/wundergraph/cosmo/router/gen/proto/wg/cosmo/node/v1"
	"github.com/wundergraph/cosmo/router/pkg/config"
	"github.com/wundergraph/cosmo/router/pkg/controlplane/configpoller"
	"github.com/wundergraph/cosmo/router/pkg/routerconfig"
)

const (
	// Two operations the test CDN knows. The first is sent as a plain body, the second by ID.
	snapshotKnownBody   = "query Employees {\n employees {\n id\n }\n}"
	snapshotKnownBodyID = "33651da3d80e420709520fb900c7ab8ec4151555da56062feeee428cf7f3a5dd"
	snapshotKnownID     = "9015ddfadd802bb378a14e48cea51e9bf9a07c7f8a71d85c56d7b104fea84937"
	snapshotKnownIDBody = "query Employees {\n  employees {\n    id\n    }\n}"
	snapshotAdHocQuery  = "query Employees {\n\n\n employees {\n id\n }\n}"
	snapshotAdHocHash   = "5e72e7c4cf0f86f7bc7044eb0c932917f3491c5f63fb769b96e5ded98c4ac0a5"
)

// snapshotClientHeader returns a new header for each request, because the test
// environment modifies the header it is given and subtests run in parallel.
func snapshotClientHeader() http.Header {
	return http.Header{"graphql-client-name": {"my-client"}}
}

// snapshotProbe records what a module can see through ModuleContext.Config and OperationContext.
type snapshotProbe struct {
	mu            sync.Mutex
	config        core.ModuleConfig
	persistedID   string
	isPersisted   bool
	rawContent    string
	sawOperation  bool
	provisionedOK bool

	// seen by SubscriptionOnStart, for operations that arrive over WebSocket
	startCalls       int
	startPersistedID string
	startIsPersisted bool
	startRawContent  string
}

type snapshotProbeModule struct {
	probe *snapshotProbe
}

func (m *snapshotProbeModule) Provision(ctx *core.ModuleContext) error {
	m.probe.mu.Lock()
	defer m.probe.mu.Unlock()
	m.probe.config = ctx.Config
	m.probe.provisionedOK = true
	return nil
}

func (m *snapshotProbeModule) Middleware(ctx core.RequestContext, next http.Handler) {
	op := ctx.Operation()
	m.probe.mu.Lock()
	m.probe.persistedID = op.PersistedID()
	m.probe.isPersisted = op.IsPersisted()
	m.probe.rawContent = op.RawContent()
	m.probe.sawOperation = true
	m.probe.mu.Unlock()
	next.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
}

func (m *snapshotProbeModule) SubscriptionOnStart(ctx core.SubscriptionOnStartHandlerContext) error {
	op := ctx.Operation()
	m.probe.mu.Lock()
	defer m.probe.mu.Unlock()
	m.probe.startCalls++
	m.probe.startPersistedID = op.PersistedID()
	m.probe.startIsPersisted = op.IsPersisted()
	m.probe.startRawContent = op.RawContent()
	return nil
}

func (m *snapshotProbeModule) Module() core.ModuleInfo {
	return core.ModuleInfo{
		ID:       "snapshotProbeModule",
		Priority: 1,
		New:      func() core.Module { return &snapshotProbeModule{probe: m.probe} },
	}
}

var (
	_ core.Provisioner                = (*snapshotProbeModule)(nil)
	_ core.RouterMiddlewareHandler    = (*snapshotProbeModule)(nil)
	_ core.SubscriptionOnStartHandler = (*snapshotProbeModule)(nil)
)

func (p *snapshotProbe) operation() (id string, persisted bool, raw string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.persistedID, p.isPersisted, p.rawContent
}

func (p *snapshotProbe) moduleConfig() core.ModuleConfig {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.config
}

func TestOperationIdentity(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, opts []core.Option, fn func(t *testing.T, xEnv *testenv.Environment, probe *snapshotProbe)) {
		t.Helper()
		probe := &snapshotProbe{}
		testenv.Run(t, &testenv.Config{
			RouterOptions: append([]core.Option{core.WithCustomModules(&snapshotProbeModule{probe: probe})}, opts...),
		}, func(t *testing.T, xEnv *testenv.Environment) {
			fn(t, xEnv, probe)
		})
	}

	t.Run("persisted operation sent by ID", func(t *testing.T) {
		t.Parallel()
		run(t, nil, func(t *testing.T, xEnv *testenv.Environment, probe *snapshotProbe) {
			xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				OperationName: []byte(`"Employees"`),
				Extensions:    []byte(`{"persistedQuery": {"version": 1, "sha256Hash": "` + snapshotKnownID + `"}}`),
				Header:        snapshotClientHeader(),
			})
			id, persisted, raw := probe.operation()
			require.True(t, persisted)
			require.Equal(t, snapshotKnownID, id)
			require.Equal(t, snapshotKnownIDBody, raw)
		})
	})

	t.Run("ad hoc query is not persisted and has no ID", func(t *testing.T) {
		t.Parallel()
		run(t, nil, func(t *testing.T, xEnv *testenv.Environment, probe *snapshotProbe) {
			xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				OperationName: []byte(`"Employees"`),
				Query:         snapshotAdHocQuery,
				Header:        snapshotClientHeader(),
			})
			id, persisted, raw := probe.operation()
			require.False(t, persisted)
			require.Empty(t, id)
			require.Equal(t, snapshotAdHocQuery, raw)
		})
	})

	t.Run("ad hoc query with log_unknown gets a computed ID but is not persisted", func(t *testing.T) {
		t.Parallel()
		run(t, []core.Option{
			core.WithPersistedOperationsConfig(config.PersistedOperationsConfig{LogUnknown: true}),
		}, func(t *testing.T, xEnv *testenv.Environment, probe *snapshotProbe) {
			xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				OperationName: []byte(`"Employees"`),
				Query:         snapshotAdHocQuery,
				Header:        snapshotClientHeader(),
			})
			id, persisted, _ := probe.operation()
			require.False(t, persisted)
			require.Equal(t, snapshotAdHocHash, id)
		})
	})

	t.Run("known query sent as a plain body with safelist is persisted", func(t *testing.T) {
		t.Parallel()
		run(t, []core.Option{
			core.WithPersistedOperationsConfig(config.PersistedOperationsConfig{
				Safelist: config.SafelistConfiguration{Enabled: true},
			}),
		}, func(t *testing.T, xEnv *testenv.Environment, probe *snapshotProbe) {
			xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				OperationName: []byte(`"Employees"`),
				Query:         snapshotKnownBody,
				Header:        snapshotClientHeader(),
			})
			id, persisted, _ := probe.operation()
			require.True(t, persisted)
			require.Equal(t, snapshotKnownBodyID, id)
		})
	})
}

func TestOperationIdentityOverWebSocket(t *testing.T) {
	t.Parallel()

	const (
		subscriptionID   = "6e94d99132b544a0d7522696a7d35643d56a26c7b8c2e0df29e2b9935636628c"
		subscriptionBody = `subscription { currentTime { unixTime timeStamp } }`
	)

	run := func(t *testing.T, payload string) *snapshotProbe {
		t.Helper()
		probe := &snapshotProbe{}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{core.WithCustomModules(&snapshotProbeModule{probe: probe})},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			conn := xEnv.InitGraphQLWebSocketConnection(nil, nil, []byte(`{"graphql-client-name": "my-client"}`))
			require.NoError(t, testenv.WSWriteJSON(t, conn, testenv.WebSocketMessage{
				ID:      "1",
				Type:    "subscribe",
				Payload: []byte(payload),
			}))
			var res testenv.WebSocketMessage
			require.NoError(t, testenv.WSReadJSON(t, conn, &res))
			require.Equal(t, "next", res.Type)

			require.Eventually(t, func() bool {
				probe.mu.Lock()
				defer probe.mu.Unlock()
				return probe.startCalls >= 1
			}, 10*time.Second, 50*time.Millisecond)
			require.NoError(t, conn.Close())
		})
		return probe
	}

	t.Run("persisted subscription sent by ID", func(t *testing.T) {
		t.Parallel()
		probe := run(t, `{"extensions":{"persistedQuery":{"version":1,"sha256Hash":"`+subscriptionID+`"}}}`)
		probe.mu.Lock()
		defer probe.mu.Unlock()
		require.True(t, probe.startIsPersisted)
		require.Equal(t, subscriptionID, probe.startPersistedID)
		require.NotEmpty(t, probe.startRawContent)
	})

	t.Run("ad hoc subscription is not persisted", func(t *testing.T) {
		t.Parallel()
		probe := run(t, `{"query":"`+subscriptionBody+`"}`)
		probe.mu.Lock()
		defer probe.mu.Unlock()
		require.False(t, probe.startIsPersisted)
		require.Empty(t, probe.startPersistedID)
		require.Equal(t, subscriptionBody, probe.startRawContent)
	})
}

func TestModulePersistedOperationManifest(t *testing.T) {
	t.Parallel()

	manifestConfig := config.PersistedOperationsConfig{
		Manifest: config.PQLManifestConfig{
			Enabled:      true,
			PollInterval: 10 * time.Second,
			PollJitter:   5 * time.Second,
		},
	}

	t.Run("exposes the approved manifest", func(t *testing.T) {
		t.Parallel()
		probe := &snapshotProbe{}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{
				core.WithPersistedOperationsConfig(manifestConfig),
				core.WithCustomModules(&snapshotProbeModule{probe: probe}),
			},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			manifest := probe.moduleConfig().PersistedOperations.Manifest()
			require.NotNil(t, manifest)
			require.Positive(t, manifest.Len())

			body, ok := manifest.Lookup(snapshotKnownID)
			require.True(t, ok)
			require.Equal(t, snapshotKnownIDBody, body)

			_, ok = manifest.Lookup(snapshotAdHocHash)
			require.False(t, ok)

			ranged := 0
			manifest.Range(func(id, body string) bool {
				ranged++
				return false // stops after the first entry
			})
			require.Equal(t, 1, ranged)
		})
	})

	t.Run("a nil manifest is safe to use", func(t *testing.T) {
		t.Parallel()
		var manifest *core.PersistedOperationManifest
		_, ok := manifest.Lookup("x")
		require.False(t, ok)
		require.Zero(t, manifest.Len())
		manifest.Range(func(string, string) bool { t.Fatal("unreachable"); return true })
	})

	t.Run("is nil when no manifest is configured", func(t *testing.T) {
		t.Parallel()
		probe := &snapshotProbe{}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{core.WithCustomModules(&snapshotProbeModule{probe: probe})},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			require.Nil(t, probe.moduleConfig().PersistedOperations.Manifest())
		})
	})

	t.Run("an APQ-registered operation reports persisted but is not in the manifest", func(t *testing.T) {
		t.Parallel()
		probe := &snapshotProbe{}
		testenv.Run(t, &testenv.Config{
			ApqConfig: config.AutomaticPersistedQueriesConfig{
				Enabled: true,
				Cache:   config.AutomaticPersistedQueriesCacheConfig{Size: 1024 * 1024},
			},
			RouterOptions: []core.Option{
				core.WithPersistedOperationsConfig(manifestConfig),
				core.WithCustomModules(&snapshotProbeModule{probe: probe}),
			},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				OperationName: []byte(`"Employees"`),
				Extensions:    []byte(`{"persistedQuery": {"version": 1, "sha256Hash": "` + snapshotAdHocHash + `"}}`),
				Query:         snapshotAdHocQuery,
				Header:        snapshotClientHeader(),
			})
			id, persisted, _ := probe.operation()
			require.True(t, persisted)
			require.Equal(t, snapshotAdHocHash, id)

			_, approved := probe.moduleConfig().PersistedOperations.Manifest().Lookup(id)
			require.False(t, approved)
		})
	})
}

// reloadPoller is a config poller whose updates the test triggers by hand.
type reloadPoller struct {
	initConfig *nodev1.RouterConfig
	handler    func(*routerconfig.Response) error
	ready      chan struct{}
	once       sync.Once
}

func (p *reloadPoller) Subscribe(_ context.Context, handler func(*routerconfig.Response) error) {
	p.handler = handler
	p.once.Do(func() { close(p.ready) })
}
func (p *reloadPoller) GetRouterConfig(context.Context) (*routerconfig.Response, error) {
	return &routerconfig.Response{Config: p.initConfig}, nil
}
func (p *reloadPoller) Stop(context.Context) error { return nil }

var _ configpoller.ConfigPoller = (*reloadPoller)(nil)

func TestModuleClientSchema(t *testing.T) {
	t.Parallel()

	t.Run("base and feature flag graphs have their own schema", func(t *testing.T) {
		t.Parallel()
		probe := &snapshotProbe{}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{core.WithCustomModules(&snapshotProbeModule{probe: probe})},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			schema := probe.moduleConfig().ClientSchema

			base := schema("")
			require.NotNil(t, base)
			require.NotEmpty(t, base.SDL())
			require.NotEmpty(t, base.Digest)
			require.NotContains(t, base.SDL(), "productCount")

			ff := schema("myff")
			require.NotNil(t, ff)
			require.Contains(t, ff.SDL(), "productCount")
			require.NotEqual(t, base.Digest, ff.Digest)

			require.Nil(t, schema("nonexistent"))
		})
	})

	t.Run("a nil schema is safe to use", func(t *testing.T) {
		t.Parallel()
		var s *core.SchemaSnapshot
		require.Empty(t, s.SDL())
	})

	t.Run("schema follows a config reload", func(t *testing.T) {
		t.Parallel()
		probe := &snapshotProbe{}
		poller := &reloadPoller{ready: make(chan struct{})}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{core.WithCustomModules(&snapshotProbeModule{probe: probe})},
			RouterConfig: &testenv.RouterConfig{
				ConfigPollerFactory: func(cfg *nodev1.RouterConfig) configpoller.ConfigPoller {
					poller.initConfig = cfg
					return poller
				},
			},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			schema := probe.moduleConfig().ClientSchema
			before := schema("")
			require.NotNil(t, before)
			require.Equal(t, xEnv.RouterConfigVersionMain(), before.Revision)

			<-poller.ready
			poller.initConfig.Version = "updated"
			require.NoError(t, poller.handler(&routerconfig.Response{Config: poller.initConfig}))

			after := schema("")
			require.NotNil(t, after)
			require.Equal(t, "updated", after.Revision)
			require.Equal(t, before.Digest, after.Digest, "same SDL, same digest")
			require.NotNil(t, schema("myff"), "feature flag schema survives the reload")
		})
	})
}
