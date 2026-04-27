package codegen

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Fel1xKan/axle/pkg/axle"
)

var routeParamRE = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// BuildRoutes converts descriptor operations into explicit runtime routes.
func BuildRoutes(desc axle.Descriptor) []axle.RouteDescriptor {
	base := "/" + strings.Trim(desc.Resource.Path, "/")
	var routes []axle.RouteDescriptor
	for _, op := range desc.Resource.Operations {
		routes = append(routes, routeForOperation(base, desc.Resource.ID, op))
	}
	for _, op := range desc.Resource.Actions {
		path := base + "/{" + desc.Resource.ID + "}/" + strings.Trim(op.Path, "/")
		routes = append(routes, routeFromOperation(op, "POST", path))
	}
	sort.SliceStable(routes, func(i, j int) bool { return routes[i].Path+routes[i].Name < routes[j].Path+routes[j].Name })
	return routes
}

func routeForOperation(base, id string, op axle.OperationDescriptor) axle.RouteDescriptor {
	switch op.Kind {
	case "list":
		return routeFromOperation(op, "GET", base)
	case "get":
		return routeFromOperation(op, "GET", base+"/{"+id+"}")
	case "create":
		return routeFromOperation(op, "POST", base)
	case "update":
		return routeFromOperation(op, "POST", base+"/{"+id+"}/update")
	case "delete":
		return routeFromOperation(op, "POST", base+"/{"+id+"}/delete")
	default:
		return routeFromOperation(op, "POST", base+"/{"+id+"}/"+strings.Trim(op.Path, "/"))
	}
}

func routeFromOperation(op axle.OperationDescriptor, method, path string) axle.RouteDescriptor {
	matches := routeParamRE.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		params = append(params, match[1])
	}
	return axle.RouteDescriptor{
		Name:            op.Name,
		Kind:            op.Kind,
		TransportMethod: method,
		Path:            path,
		Params:          params,
		Request:         op.Request,
		Response:        op.Response,
		Policy:          op.Policy,
		Handler:         op.Handler,
	}
}
