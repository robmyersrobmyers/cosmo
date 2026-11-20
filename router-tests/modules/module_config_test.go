package module_test

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wundergraph/cosmo/router-tests/testenv"
	"github.com/wundergraph/cosmo/router/core"
	"github.com/wundergraph/cosmo/router/pkg/config"
)

var safelistEnabledSeen atomic.Int32 // 0 = not provisioned, 1 = false, 2 = true

type configProbeModule struct{}

func (m *configProbeModule) Provision(ctx *core.ModuleContext) error {
	if ctx.Config.PersistedOperations.Safelist.Enabled {
		safelistEnabledSeen.Store(2)
	} else {
		safelistEnabledSeen.Store(1)
	}
	return nil
}

func (m *configProbeModule) Module() core.ModuleInfo {
	return core.ModuleInfo{
		ID:  "configProbeModule",
		New: func() core.Module { return &configProbeModule{} },
	}
}

var _ core.Provisioner = (*configProbeModule)(nil)

func TestModuleContextConfig(t *testing.T) {
	// Not parallel: the probe module reports through a package-level variable.
	for _, tc := range []struct {
		name     string
		safelist bool
		want     int32
	}{
		{name: "safelist disabled", safelist: false, want: 1},
		{name: "safelist enabled", safelist: true, want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			safelistEnabledSeen.Store(0)
			testenv.Run(t, &testenv.Config{
				RouterOptions: []core.Option{
					core.WithPersistedOperationsConfig(config.PersistedOperationsConfig{
						Safelist: config.SafelistConfiguration{Enabled: tc.safelist},
					}),
					core.WithCustomModules(&configProbeModule{}),
				},
			}, func(t *testing.T, xEnv *testenv.Environment) {
				require.Equal(t, tc.want, safelistEnabledSeen.Load())
			})
		})
	}
}
