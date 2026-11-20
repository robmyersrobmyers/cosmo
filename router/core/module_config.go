package core

// This file holds what modules can read from the router through ModuleContext.Config.
// It is kept apart from modules.go and router.go on purpose, so that changes to those
// files upstream rarely conflict with it. Only one line in each of them refers to it.

// ModuleConfig exposes selected router settings to modules.
// Settings are added here as modules need them.
type ModuleConfig struct {
	PersistedOperations ModulePersistedOperationsConfig

	// ClientSchema returns the client schema of the active graph for the given feature flag.
	// The empty name is the base graph. It returns nil for an unknown feature flag and before the
	// first graph server is running, for example during Provision. After a config reload it
	// returns the new schema, so call it when needed and don't keep the result.
	ClientSchema func(featureFlag string) *SchemaSnapshot
}

// ModulePersistedOperationsConfig exposes the router's persisted operations settings to modules.
type ModulePersistedOperationsConfig struct {
	// Safelist mirrors persisted_operations.safelist.
	Safelist ModuleSafelistConfig

	// Manifest returns the current approved persisted operation manifest. It returns nil when no
	// manifest is in use (operations come from a storage provider) or has not been loaded yet.
	// After a manifest reload it returns the new manifest, so call it when needed and don't keep the result.
	Manifest func() *PersistedOperationManifest
}

// ModuleSafelistConfig exposes the router's persisted operations safelist setting.
type ModuleSafelistConfig struct {
	// Enabled is true when the router rejects every operation that is not in the persisted operation store.
	Enabled bool
}

// moduleContextConfig builds the ModuleConfig that every module receives in its ModuleContext.
func (r *Router) moduleContextConfig() ModuleConfig {
	return ModuleConfig{
		PersistedOperations: ModulePersistedOperationsConfig{
			Safelist: ModuleSafelistConfig{
				Enabled: r.persistedOperationsConfig.Safelist.Enabled,
			},
			Manifest: r.persistedOperationManifest,
		},
		ClientSchema: r.clientSchema,
	}
}

// persistedOperationManifest returns the current approved persisted operation manifest, or nil.
func (r *Router) persistedOperationManifest() *PersistedOperationManifest {
	return newPersistedOperationManifest(r.persistedOperationClient.ManifestSnapshot())
}

// clientSchema returns the client schema of the active graph server for a feature flag, or nil.
// The empty name is the base graph.
func (r *Router) clientSchema(featureFlag string) *SchemaSnapshot {
	if r.httpServer == nil {
		return nil
	}
	state := r.httpServer.state.Load()
	if state == nil || state.graphServer == nil {
		return nil
	}

	gs := state.graphServer
	gs.graphMuxListLock.Lock()
	defer gs.graphMuxListLock.Unlock()

	gm := gs.graphMuxList[featureFlag]
	if gm == nil {
		return nil
	}
	return gm.clientSchema
}
