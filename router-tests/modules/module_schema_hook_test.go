package module_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wundergraph/cosmo/router-tests/testenv"
	"github.com/wundergraph/cosmo/router/core"
	nodev1 "github.com/wundergraph/cosmo/router/gen/proto/wg/cosmo/node/v1"
	"github.com/wundergraph/cosmo/router/pkg/controlplane/configpoller"
	"github.com/wundergraph/cosmo/router/pkg/routerconfig"
)

type schemaHookCall struct {
	featureFlag string
	revision    string
	sdl         string
}

type schemaHookModule struct {
	mu    sync.Mutex
	calls []schemaHookCall
}

func (m *schemaHookModule) Module() core.ModuleInfo {
	return core.ModuleInfo{ID: "schemaHookModule", New: func() core.Module { return m }}
}

func (m *schemaHookModule) OnClientSchema(featureFlag string, schema *core.SchemaSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, schemaHookCall{featureFlag, schema.Revision, schema.SDL()})
	return nil
}

func (m *schemaHookModule) snapshot() []schemaHookCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]schemaHookCall(nil), m.calls...)
}

func TestClientSchemaHandler(t *testing.T) {
	t.Parallel()

	t.Run("called at startup for the base and feature flag graphs", func(t *testing.T) {
		t.Parallel()
		mod := &schemaHookModule{}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{core.WithCustomModules(mod)},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			calls := mod.snapshot()
			require.Len(t, calls, 2)
			require.Equal(t, "", calls[0].featureFlag, "base graph first")
			require.Equal(t, "myff", calls[1].featureFlag)
			require.NotContains(t, calls[0].sdl, "productCount")
			require.Contains(t, calls[1].sdl, "productCount")
			require.Equal(t, xEnv.RouterConfigVersionMain(), calls[0].revision)
		})
	})

	t.Run("called again after a config reload", func(t *testing.T) {
		t.Parallel()
		mod := &schemaHookModule{}
		poller := &reloadPoller{ready: make(chan struct{})}
		testenv.Run(t, &testenv.Config{
			RouterOptions: []core.Option{core.WithCustomModules(mod)},
			RouterConfig: &testenv.RouterConfig{
				ConfigPollerFactory: func(cfg *nodev1.RouterConfig) configpoller.ConfigPoller {
					poller.initConfig = cfg
					return poller
				},
			},
		}, func(t *testing.T, xEnv *testenv.Environment) {
			startupCalls := len(mod.snapshot())
			require.NotZero(t, startupCalls)

			<-poller.ready
			poller.initConfig.Version = "updated"
			require.NoError(t, poller.handler(&routerconfig.Response{Config: poller.initConfig}))

			calls := mod.snapshot()
			require.Greater(t, len(calls), startupCalls)
			require.Equal(t, "updated", calls[startupCalls].revision)
		})
	})
}
