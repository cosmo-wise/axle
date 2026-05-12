# Harness Integration Example

This example shows how Axle descriptors integrate with Harness
for automated backend generation.

## Setup

1. Place your `.axle.json` descriptor files in the target repo
2. Configure Harness to use the Axle generator
3. Run: `axle gen && harness run --input .`

## Descriptor Example

```json
{
  "apiVersion": "axle/v1",
  "kind": "Resource",
  "metadata": { "name": "Task" },
  "spec": {
    "fields": [
      { "name": "title", "type": "string", "validate": "required" },
      { "name": "done", "type": "boolean", "default": false }
    ]
  }
}
```
