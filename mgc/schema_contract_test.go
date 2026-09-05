package mgc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Freeze the published wire schema, not descriptions or implementation details.
// An intentional change requires a reviewed baseline update and, where needed,
// state migration tests. This is complementary to real Terraform plan tests:
// RequiresReplace and other plan modifiers are not exposed by GetProviderSchema.
func TestProviderSchemaContract(t *testing.T) {
	server := providerserver.NewProtocol6(New("schema-contract")())()
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range response.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("invalid provider schema: %s: %s", d.Summary, d.Detail)
		}
	}
	var lines []string
	add := func(prefix string, schema *tfprotov6.Schema) {
		lines = append(lines, fmt.Sprintf("%s version=%d", prefix, schema.Version))
		lines = append(lines, schemaBlockContract(prefix, schema.Block)...)
	}
	add("provider", response.Provider)
	for name, schema := range response.ResourceSchemas {
		add("resource."+name, schema)
	}
	for name, schema := range response.DataSourceSchemas {
		add("data."+name, schema)
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"
	const filename = "testdata/schema-contract.golden"
	if os.Getenv("UPDATE_SCHEMA_CONTRACT") == "1" {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(strings.Split(string(want), "\n"), strings.Split(got, "\n")); diff != "" {
		t.Fatalf("published schema changed (-baseline +current); review compatibility and migration before updating testdata/schema-contract.golden:\n%s", diff)
	}
}

func schemaBlockContract(prefix string, block *tfprotov6.SchemaBlock) []string {
	if block == nil {
		return nil
	}
	lines := schemaAttributesContract(prefix, block.Attributes)
	for _, nested := range block.BlockTypes {
		path := prefix + "." + nested.TypeName
		lines = append(lines, fmt.Sprintf("%s block=%d min=%d max=%d", path, nested.Nesting, nested.MinItems, nested.MaxItems))
		lines = append(lines, schemaBlockContract(path, nested.Block)...)
	}
	return lines
}

func schemaAttributesContract(prefix string, attributes []*tfprotov6.SchemaAttribute) []string {
	var lines []string
	for _, attribute := range attributes {
		path := prefix + "." + attribute.Name
		var kind string
		if attribute.NestedType != nil {
			kind = fmt.Sprintf("nested:%d", attribute.NestedType.Nesting)
			lines = append(lines, schemaAttributesContract(path, attribute.NestedType.Attributes)...)
		} else {
			encoded, err := json.Marshal(attribute.Type)
			if err != nil {
				panic(err)
			} // A schema type must be JSON serializable.
			kind = string(encoded)
		}
		lines = append(lines, fmt.Sprintf("%s type=%s required=%t optional=%t computed=%t sensitive=%t write_only=%t", path, kind, attribute.Required, attribute.Optional, attribute.Computed, attribute.Sensitive, attribute.WriteOnly))
	}
	return lines
}
