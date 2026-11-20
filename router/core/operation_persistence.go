package core

// setPersistence records how the router identified an operation as persisted. Both the HTTP
// handler and the WebSocket handler build their own operationContext, so both call this after the
// persisted operation was resolved. A path that builds an operationContext without calling it
// reports every operation as not persisted.
//
// persistedID is the ID the request supplied. With safelist or log_unknown enabled the router also
// fills it with the computed SHA-256 of a plain query, so isPersisted must be read separately.
func (o *operationContext) setPersistence(parsed *ParsedOperation) {
	o.isPersisted = parsed.IsPersistedOperation
	if pq := parsed.GraphQLRequestExtensions.PersistedQuery; pq.HasHash() {
		o.persistedID = pq.Sha256Hash
	}
}
