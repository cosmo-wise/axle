package runtime

import (
	"net/http"
	"strings"

	"github.com/Fel1xKan/axle/pkg/axle"
	axlesqlite "github.com/Fel1xKan/axle/pkg/axle/sqlite"
)

// EdgeOptions configures Axle's optional application edge wrapper.
//
// The wrapper keeps common app-owned conveniences out of project code while
// leaving generated CRUD and custom actions in the descriptor-driven runtime.
type EdgeOptions struct {
	// Name is returned from GET / and helps generated backends identify
	// themselves without hand-written route switches.
	Name string
	// APIPrefix allows the same generated runtime to answer under a stable
	// version prefix, for example /api/v1/tasks.
	APIPrefix string
	// CORS enables permissive GET/POST/OPTIONS headers for static frontend demos.
	CORS bool
	// Health is merged into the default /healthz payload.
	Health map[string]any
}

// NewEdge mounts a generated catalog with CRUD/action runtime behavior plus
// common thin edge conveniences: /healthz, /routes, optional CORS, and optional
// API prefix normalization.
func NewEdge(catalog axle.Catalog, db *axlesqlite.Database, handlers ActionHandlers, options EdgeOptions) http.Handler {
	return edgeHandler{
		runtime: New(catalog, db, handlers),
		catalog: catalog,
		options: options,
	}
}

type edgeHandler struct {
	runtime http.Handler
	catalog axle.Catalog
	options EdgeOptions
}

func (h edgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.options.CORS {
		setEdgeCORSHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	path := normalizeEdgePath(r.URL.Path, h.options.APIPrefix)
	switch path {
	case "/healthz":
		writeJSON(w, http.StatusOK, h.healthPayload())
		return
	case "/routes":
		routes := routeCatalog(h.catalog)
		writeJSON(w, http.StatusOK, map[string]any{"data": routes, "count": len(routes)})
		return
	case "/":
		writeJSON(w, http.StatusOK, map[string]any{"name": edgeName(h.options), "framework": "axle", "routes": "/routes", "health": "/healthz"})
		return
	}

	forward := r.Clone(r.Context())
	urlCopy := *r.URL
	urlCopy.Path = path
	forward.URL = &urlCopy
	h.runtime.ServeHTTP(w, forward)
}

func (h edgeHandler) healthPayload() map[string]any {
	payload := map[string]any{"status": "ok", "framework": "axle", "storage": "sqlite"}
	for key, value := range h.options.Health {
		payload[key] = value
	}
	return payload
}

func edgeName(options EdgeOptions) string {
	if strings.TrimSpace(options.Name) == "" {
		return "Axle backend"
	}
	return options.Name
}

func routeCatalog(catalog axle.Catalog) []map[string]any {
	out := []map[string]any{}
	for _, registry := range catalog.Resources {
		for _, route := range registry.Routes {
			out = append(out, map[string]any{
				"resource": registry.Resource.Path,
				"name":     route.Name,
				"kind":     route.Kind,
				"method":   route.TransportMethod,
				"path":     route.Path,
				"handler":  route.Handler,
			})
		}
	}
	return out
}

func normalizeEdgePath(rawPath string, apiPrefix string) string {
	cleaned := "/" + strings.Trim(rawPath, "/")
	if cleaned == "/" {
		return "/"
	}
	prefix := "/" + strings.Trim(apiPrefix, "/")
	if prefix != "/" {
		if cleaned == prefix {
			return "/"
		}
		if strings.HasPrefix(cleaned, prefix+"/") {
			cleaned = strings.TrimPrefix(cleaned, prefix)
		}
	}
	return strings.TrimRight(cleaned, "/")
}

func setEdgeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
