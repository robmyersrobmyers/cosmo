package core

import (
	"fmt"
	"sort"
)

// This file holds the client schema hook for modules. It is kept apart from router.go and
// graph_server.go on purpose, so that upstream changes to those files rarely conflict with it.
// Only one call in graph_server.go refers to it.

// ClientSchemaHandler lets a module react when the router builds a client schema, for example to
// pre-compute data from it at startup.
//
// OnClientSchema is called once per newly built graph (the base graph and each feature flag graph)
// while the router starts and on every config reload that rebuilds a graph. A graph that a reload
// reuses unchanged is not reported again. The empty feature flag is the base graph.
//
// The call happens before the new graph server takes traffic, and ModuleConfig.ClientSchema still
// returns the previous schema (or nil on startup). Use the schema argument instead.
//
// Returning an error stops the router from starting. On a config reload it rejects the new config,
// and the router keeps serving the previous one. The handler runs on the startup or reload goroutine,
// so keep it fast and don't block.
type ClientSchemaHandler interface {
	OnClientSchema(featureFlag string, schema *SchemaSnapshot) error
}

// notifyClientSchemaHandlers reports the client schema of every graph that s newly built to the modules.
// Graphs are reported in feature flag name order, with the base graph first.
func (r *Router) notifyClientSchemaHandlers(s *graphServer) error {
	var handlers []ClientSchemaHandler
	for _, module := range r.modules {
		if h, ok := module.(ClientSchemaHandler); ok {
			handlers = append(handlers, h)
		}
	}
	if len(handlers) == 0 {
		return nil
	}

	s.graphMuxListLock.Lock()
	names := make([]string, 0, len(s.graphMuxList))
	schemas := make(map[string]*SchemaSnapshot, len(s.graphMuxList))
	for name, gm := range s.graphMuxList {
		names = append(names, name)
		schemas[name] = gm.clientSchema
	}
	s.graphMuxListLock.Unlock()
	sort.Strings(names)

	for _, name := range names {
		for _, h := range handlers {
			if err := h.OnClientSchema(name, schemas[name]); err != nil {
				return fmt.Errorf("module failed to handle client schema of feature flag '%s': %w", name, err)
			}
		}
	}
	return nil
}
