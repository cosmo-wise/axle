package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
	axlesqlite "github.com/cosmo-wise/axle/pkg/axle/sqlite"
)

// ActionRequest is the generic runtime envelope that generated typed adapters unwrap.
type ActionRequest struct {
	Resource string               `json:"resource"`
	Route    axle.RouteDescriptor `json:"route"`
	ID       string               `json:"id,omitempty"`
	Params   map[string]string    `json:"params,omitempty"`
	Body     map[string]any       `json:"body,omitempty"`
}

// ActionHandler handles a custom descriptor action.
type ActionHandler func(context.Context, ActionRequest) (any, error)

// ActionHandlers maps generated handler names to custom action implementations.
type ActionHandlers map[string]ActionHandler

// New mounts generated CRUD/action routes against the SQLite facade.
func New(catalog axle.Catalog, db *axlesqlite.Database, handlers ActionHandlers) http.Handler {
	return Handler{catalog: catalog, db: db, handlers: handlers}
}

// Handler is an explicit generated-catalog runtime mount.
type Handler struct {
	catalog  axle.Catalog
	db       *axlesqlite.Database
	handlers ActionHandlers
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathMatched := false
	var allowed []string
	for _, registry := range h.catalog.Resources {
		for _, route := range registry.Routes {
			params, ok := matchPath(route.Path, r.URL.Path)
			if !ok {
				continue
			}
			if r.Method != route.TransportMethod {
				pathMatched = true
				allowed = append(allowed, route.TransportMethod)
				continue
			}
			h.handleRoute(w, r, registry.Resource, route, params)
			return
		}
	}
	if pathMatched {
		writeError(w, http.StatusMethodNotAllowed, fmt.Sprintf("method %s not allowed; allowed: %s", r.Method, strings.Join(uniqueStrings(allowed), ", ")))
		return
	}
	writeError(w, http.StatusNotFound, "route not found")
}

func (h Handler) handleRoute(w http.ResponseWriter, r *http.Request, resource axle.ResourceDescriptor, route axle.RouteDescriptor, params map[string]string) {
	ctx := r.Context()
	id := params[resource.ID]
	switch route.Kind {
	case "list":
		items, err := h.db.List(ctx, resource.Path)
		writeResult(w, map[string]any{"data": items}, err)
	case "get":
		item, err := h.db.Get(ctx, resource.Path, id)
		writeResult(w, map[string]any{"data": item}, err)
	case "create":
		body, err := readRecord(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := h.db.Create(ctx, resource.Path, axlesqlite.Record(body))
		writeResult(w, map[string]any{"data": item}, err)
	case "update":
		body, err := readRecord(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := h.db.Update(ctx, resource.Path, id, axlesqlite.Record(body))
		writeResult(w, map[string]any{"data": item}, err)
	case "delete":
		err := h.db.Delete(ctx, resource.Path, id)
		writeResult(w, map[string]any{"deleted": err == nil, "id": id}, err)
	case "action":
		handler := h.handlers[route.Handler]
		if handler == nil {
			writeError(w, http.StatusNotImplemented, "missing action handler "+route.Handler)
			return
		}
		body, err := readOptionalRecord(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := handler(ctx, ActionRequest{Resource: resource.Path, Route: route, ID: id, Params: params, Body: body})
		writeResult(w, result, err)
	default:
		writeError(w, http.StatusNotImplemented, "unsupported route kind "+route.Kind)
	}
}

func readRecord(r *http.Request) (map[string]any, error) {
	body, err := readOptionalRecord(r)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return map[string]any{}, nil
	}
	return body, nil
}

func readOptionalRecord(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	var payload map[string]any
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, http.ErrBodyNotAllowed) || err.Error() == "EOF" {
			return nil, nil
		}
		return nil, err
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return data, nil
	}
	return payload, nil
}

func writeResult(w http.ResponseWriter, payload any, err error) {
	if err != nil {
		if axlesqlite.IsNotFound(err) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func matchPath(pattern, actual string) (map[string]string, bool) {
	patternParts := splitPath(pattern)
	actualParts := splitPath(actual)
	if len(patternParts) != len(actualParts) {
		return nil, false
	}
	params := map[string]string{}
	for i, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			params[strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")] = actualParts[i]
			continue
		}
		if part != actualParts[i] {
			return nil, false
		}
	}
	return params, true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
