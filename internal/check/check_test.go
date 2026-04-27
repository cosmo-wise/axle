package check_test

import (
	"slices"
	"testing"

	"github.com/Fel1xKan/axle/internal/check"
)

func TestCheckPositiveFixture(t *testing.T) {
	result := check.Run("../../testdata/fixtures/single/descriptor.axle.json", "../..")
	if result.Status != "ok" {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
}

func TestCheckNegativeFixtures(t *testing.T) {
	cases := []struct {
		name       string
		descriptor string
		root       string
		want       string
	}{
		{"controller db", "", "../../testdata/fixtures/negative/controller-db", "AXLE_BOUNDARY_CONTROLLER_DB"},
		{"service http", "", "../../testdata/fixtures/negative/service-http", "AXLE_BOUNDARY_SERVICE_HTTP"},
		{"missing bindings", "../../testdata/fixtures/negative/missing-bindings/descriptor.axle.json", "", "AXLE_OPERATION_REQUEST"},
		{"public import", "", "../../testdata/fixtures/negative/public-import", "AXLE_PUBLIC_IMPORT_INTERNAL"},
		{"multi db", "", "../../testdata/fixtures/negative/multidb", "AXLE_MULTIDB_ABSTRACTION"},
		{"reflection", "", "../../testdata/fixtures/negative/reflection", "AXLE_RUNTIME_DISCOVERY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := check.Run(tc.descriptor, tc.root)
			codes := make([]string, 0, len(result.Diagnostics))
			for _, diagnostic := range result.Diagnostics {
				codes = append(codes, diagnostic.Code)
			}
			if !slices.Contains(codes, tc.want) {
				t.Fatalf("missing %s in %#v", tc.want, result.Diagnostics)
			}
		})
	}
}
