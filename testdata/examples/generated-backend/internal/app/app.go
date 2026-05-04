package app

import (
	"context"
	"net/http"

	appcatalog "github.com/cosmo-wise/axle/testdata/examples/generated-backend/catalog"
	resources "github.com/cosmo-wise/axle/testdata/examples/generated-backend/descriptors/resources/generated"
	axleruntime "github.com/cosmo-wise/axle/pkg/axle/runtime"
	axlesqlite "github.com/cosmo-wise/axle/pkg/axle/sqlite"
)

func New(db *axlesqlite.Database) http.Handler {
	return axleruntime.NewEdge(appcatalog.Catalog, db, axleruntime.ActionHandlers{
		resources.HandlerRenameResource:        resources.BindRenameResource(renameResource),
		resources.HandlerUpgradeResourcePolicy: resources.BindUpgradeResourcePolicy(upgradeResourcePolicy),
	}, axleruntime.EdgeOptions{
		Name:      "Axle generated backend",
		APIPrefix: "/api/v1",
		CORS:      true,
	})
}

func renameResource(ctx context.Context, request resources.RenameResourceRequest) (resources.RenameResourceResponse, error) {
	name, _ := request.Body["name"].(string)
	return resources.RenameResourceResponse{Data: map[string]any{"id": request.ID, "name": name}}, nil
}

func upgradeResourcePolicy(ctx context.Context, request resources.UpgradeResourcePolicyRequest) (resources.UpgradeResourcePolicyResponse, error) {
	return resources.UpgradeResourcePolicyResponse{Data: map[string]any{"id": request.ID, "policy_id": request.Params["policy_id"], "upgraded": true}}, nil
}
