package cli

import "fmt"

func renderAppCatalogManifest(moduleName string) string {
	return fmt.Sprintf(`{
  "package": "catalog",
  "resources": [
    {"alias": "policies", "import": "%s/descriptors/policies/generated"},
    {"alias": "resources", "import": "%s/descriptors/resources/generated"}
  ]
}
`, moduleName, moduleName)
}

const appResourceDescriptorJSON = `{
  "schema": "axle.resource.v1",
  "resource": {
    "name": "Resource",
    "path": "resources",
    "table": "resources",
    "id": "id",
    "fields": [
      {"name": "id", "type": "text", "mutable": false},
      {"name": "name", "type": "text", "mutable": true},
      {"name": "policy_id", "type": "text", "mutable": true}
    ],
    "operations": [
      {"name": "ListResources", "kind": "list", "request": "ListResourcesRequest", "response": "ListResourcesResponse", "policy": "resource.read", "handler": "ListResources"},
      {"name": "GetResource", "kind": "get", "request": "GetResourceRequest", "response": "GetResourceResponse", "policy": "resource.read", "handler": "GetResource"},
      {"name": "CreateResource", "kind": "create", "request": "CreateResourceRequest", "response": "CreateResourceResponse", "policy": "resource.write", "handler": "CreateResource"},
      {"name": "UpdateResource", "kind": "update", "request": "UpdateResourceRequest", "response": "UpdateResourceResponse", "policy": "resource.write", "handler": "UpdateResource"},
      {"name": "DeleteResource", "kind": "delete", "request": "DeleteResourceRequest", "response": "DeleteResourceResponse", "policy": "resource.write", "handler": "DeleteResource"}
    ],
    "actions": [
      {"name": "RenameResource", "kind": "action", "path": "rename", "request": "RenameResourceRequest", "response": "RenameResourceResponse", "policy": "resource.write", "handler": "RenameResource"},
      {"name": "UpgradeResourcePolicy", "kind": "action", "path": "policy/{policy_id}/upgrade", "request": "UpgradeResourcePolicyRequest", "response": "UpgradeResourcePolicyResponse", "policy": "resource.write", "handler": "UpgradeResourcePolicy"}
    ]
  },
  "generated": {"package": "generated"}
}
`

const appPolicyDescriptorJSON = `{
  "schema": "axle.resource.v1",
  "resource": {
    "name": "Policy",
    "path": "policies",
    "table": "policies",
    "id": "id",
    "fields": [
      {"name": "id", "type": "text", "mutable": false},
      {"name": "name", "type": "text", "mutable": true},
      {"name": "level", "type": "integer", "mutable": true}
    ],
    "operations": [
      {"name": "ListPolicies", "kind": "list", "request": "ListPoliciesRequest", "response": "ListPoliciesResponse", "policy": "policy.read", "handler": "ListPolicies"},
      {"name": "GetPolicy", "kind": "get", "request": "GetPolicyRequest", "response": "GetPolicyResponse", "policy": "policy.read", "handler": "GetPolicy"},
      {"name": "CreatePolicy", "kind": "create", "request": "CreatePolicyRequest", "response": "CreatePolicyResponse", "policy": "policy.write", "handler": "CreatePolicy"},
      {"name": "UpdatePolicy", "kind": "update", "request": "UpdatePolicyRequest", "response": "UpdatePolicyResponse", "policy": "policy.write", "handler": "UpdatePolicy"},
      {"name": "DeletePolicy", "kind": "delete", "request": "DeletePolicyRequest", "response": "DeletePolicyResponse", "policy": "policy.write", "handler": "DeletePolicy"}
    ]
  },
  "generated": {"package": "generated"}
}
`
