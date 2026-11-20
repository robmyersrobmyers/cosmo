package core

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/wundergraph/cosmo/router/internal/operationmanifest"
)

// PersistedOperationManifest is a read-only view of the approved persisted operation
// manifest at one revision. It never changes. After a manifest reload, a new call to
// ModuleContext.Config.PersistedOperations.Manifest returns a new view.
type PersistedOperationManifest struct {
	Version     int
	Revision    string
	GeneratedAt string

	// operations maps a persisted operation ID (its SHA-256) to the operation body.
	// It is shared with the router's published manifest, which is never modified.
	operations map[string]string
}

func newPersistedOperationManifest(m *operationmanifest.Manifest) *PersistedOperationManifest {
	if m == nil {
		return nil
	}
	return &PersistedOperationManifest{
		Version:     m.Version,
		Revision:    m.Revision,
		GeneratedAt: m.GeneratedAt,
		operations:  m.Operations,
	}
}

// Lookup returns the operation body for an approved persisted operation ID.
func (m *PersistedOperationManifest) Lookup(id string) (string, bool) {
	if m == nil {
		return "", false
	}
	body, ok := m.operations[id]
	return body, ok
}

// Len returns the number of approved operations.
func (m *PersistedOperationManifest) Len() int {
	if m == nil {
		return 0
	}
	return len(m.operations)
}

// Range calls fn for every approved operation until fn returns false.
// The iteration order is not defined.
func (m *PersistedOperationManifest) Range(fn func(id, body string) bool) {
	if m == nil {
		return
	}
	for id, body := range m.operations {
		if !fn(id, body) {
			return
		}
	}
}

// SchemaSnapshot is the client schema of one graph at one config version.
type SchemaSnapshot struct {
	// Revision is the router config version the schema came from.
	Revision string
	// Digest is the hex SHA-256 of the SDL.
	Digest string

	sdl string
}

// SDL returns the client schema source. It does not include the engine's built-in
// definitions (introspection fields, built-in scalars).
func (s *SchemaSnapshot) SDL() string {
	if s == nil {
		return ""
	}
	return s.sdl
}

func newSchemaSnapshot(sdl, revision string) *SchemaSnapshot {
	sum := sha256.Sum256([]byte(sdl))
	return &SchemaSnapshot{
		Revision: revision,
		Digest:   hex.EncodeToString(sum[:]),
		sdl:      sdl,
	}
}
