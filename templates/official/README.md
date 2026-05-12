# Official Axle Templates

This directory contains official code generation templates for Axle.
Templates define how resources are rendered into Go source files,
SQL migrations, OpenAPI specs, and route registrations.

Templates follow the Go `text/template` format and receive
a parsed `ResourceDescriptor` as their data context.
