package main

import (
	"os"
	"reflect"
	"testing"
)

func strPtr(s string) *string { return &s }

func createTempJSONFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "commands-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.WriteString(content)
	f.Close()
	return f.Name()
}

func TestParseType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"String", "string"},
		{"Number", "number"},
		{"Boolean", "bool"},
		{"List of String", "array(string)"},
		{"List of Number", "array(number)"},
		{"Set of String", "set(string)"},
		{"Set of Number", "set(number)"},
		{"Map of String", "map(string)"},
		{"Map of Number", "map(number)"},
		{"Attributes List", "array(object)"},
		{"Attributes", "object"},
		{"Attributes Set", "set(object)"},
		{"Map", "map"},
		{"String, Optional", "string"},
		{"Boolean, Deprecated", "bool"},
		{"UnknownType", "unknowntype"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseType(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractSection(t *testing.T) {
	content := "## Schema\n\n### Required\n\n- `id` (String) ID.\n\n### Optional\n\n- `name` (String) Name.\n\n## Import\n"

	tests := []struct {
		name   string
		header string
		stops  []string
		want   string
	}{
		{
			name:   "finds section and stops at next subsection",
			header: "### Required",
			stops:  []string{"\n### "},
			want:   "\n- `id` (String) ID.\n",
		},
		{
			name:   "stops at earliest of multiple stops",
			header: "### Optional",
			stops:  []string{"\n### ", "\n## "},
			want:   "\n- `name` (String) Name.\n",
		},
		{
			name:   "returns empty when header not found",
			header: "### Read-Only",
			stops:  []string{"\n### "},
			want:   "",
		},
		{
			name:   "returns empty when nothing follows the header",
			header: "## Import",
			stops:  []string{"\n## "},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSection(content, tt.header, tt.stops)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoveSeeBelow(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes reference with preceding text",
			input: "Vulnerability counts (see [below for nested schema](#nestedatt--severity_summary))",
			want:  "Vulnerability counts",
		},
		{
			name:  "removes standalone reference leaving empty string",
			input: "(see [below for nested schema](#nestedatt--foo))",
			want:  "",
		},
		{
			name:  "leaves text unchanged when no reference present",
			input: "No nested schema here",
			want:  "No nested schema here",
		},
		{
			name:  "removes reference between surrounding text",
			input: "Text before (see [below for nested schema](#nestedatt--bar)) text after",
			want:  "Text before text after",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeSeeBelow(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAttrLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantName    string
		wantTypeStr string
		wantDesc    string
		wantOK      bool
	}{
		{
			name:        "simple attribute with description",
			line:        "- `id` (String) ID of instance.",
			wantName:    "id",
			wantTypeStr: "String",
			wantDesc:    "ID of instance.",
			wantOK:      true,
		},
		{
			name:        "type with nested parens (write-only link)",
			line:        "- `password` (String, [Write-only](https://example.com)) Database password.",
			wantName:    "password",
			wantTypeStr: "String, [Write-only](https://example.com)",
			wantDesc:    "Database password.",
			wantOK:      true,
		},
		{
			name:        "attribute with no description",
			line:        "- `status` (String)",
			wantName:    "status",
			wantTypeStr: "String",
			wantDesc:    "",
			wantOK:      true,
		},
		{
			name:   "not an attribute line",
			line:   "Some random line",
			wantOK: false,
		},
		{
			name:   "missing type parens",
			line:   "- `name` no parens here",
			wantOK: false,
		},
		{
			name:   "unclosed paren",
			line:   "- `name` (String",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, typeStr, desc, ok := parseAttrLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("ok: got %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
			if typeStr != tt.wantTypeStr {
				t.Errorf("typeStr: got %q, want %q", typeStr, tt.wantTypeStr)
			}
			if desc != tt.wantDesc {
				t.Errorf("desc: got %q, want %q", desc, tt.wantDesc)
			}
		})
	}
}

func TestParseAttrSection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		required bool
		readOnly bool
		want     []Attribute
	}{
		{
			name:     "single required attribute",
			content:  "\n- `id` (String) ID of machine-type.\n",
			required: true,
			readOnly: false,
			want: []Attribute{
				{Name: "id", Description: "ID of machine-type.", OriginalDescription: "ID of machine-type.", Type: "string", Required: true},
			},
		},
		{
			name:     "multiple read-only attributes",
			content:  "\n- `name` (String) Name.\n- `age` (Number) Age.\n",
			required: false,
			readOnly: true,
			want: []Attribute{
				{Name: "name", Description: "Name.", OriginalDescription: "Name.", Type: "string", ReadOnly: true},
				{Name: "age", Description: "Age.", OriginalDescription: "Age.", Type: "number", ReadOnly: true},
			},
		},
		{
			name:     "section header labels are not appended as description",
			content:  "\n- `preferred` (Boolean)\nRead-Only:\n- `score` (Number) Score.\n",
			required: false,
			readOnly: true,
			want: []Attribute{
				{Name: "preferred", Description: "", OriginalDescription: "", Type: "bool", ReadOnly: true},
				{Name: "score", Description: "Score.", OriginalDescription: "Score.", Type: "number", ReadOnly: true},
			},
		},
		{
			name:     "multi-line description is joined",
			content:  "\n- `desc_attr` (String) First line.\nSecond line.\n",
			required: false,
			readOnly: false,
			want: []Attribute{
				{Name: "desc_attr", Description: "First line. Second line.", OriginalDescription: "First line. Second line.", Type: "string"},
			},
		},
		{
			name:     "removes see-below reference from description",
			content:  "\n- `interfaces` (Attributes List) Network interfaces (see [below for nested schema](#nestedatt--interfaces))\n",
			required: false,
			readOnly: true,
			want: []Attribute{
				{Name: "interfaces", Description: "Network interfaces", OriginalDescription: "Network interfaces", Type: "array(object)", ReadOnly: true},
			},
		},
		{
			name:     "skips lines starting with special markdown characters",
			content:  "\n- `name` (String) Name.\n> blockquote\n| table |\n# heading\nContinuation.\n",
			required: false,
			readOnly: false,
			want: []Attribute{
				{Name: "name", Description: "Name. Continuation.", OriginalDescription: "Name. Continuation.", Type: "string"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAttrSection(tt.content, tt.required, tt.readOnly)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseNestedSchemas(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string][]Attribute
	}{
		{
			name: "parses top-level nested schema",
			content: "<a id=\"nestedatt--interfaces\"></a>\n" +
				"### Nested Schema for `interfaces`\n\n" +
				"Read-Only:\n\n" +
				"- `id` (String) Interface ID.\n" +
				"- `name` (String) Interface name.\n",
			want: map[string][]Attribute{
				"interfaces": {
					{Name: "id", Description: "Interface ID.", OriginalDescription: "Interface ID.", Type: "string", ReadOnly: true},
					{Name: "name", Description: "Interface name.", OriginalDescription: "Interface name.", Type: "string", ReadOnly: true},
				},
			},
		},
		{
			name: "parses dotted nested path",
			content: "<a id=\"nestedatt--interfaces--security_groups\"></a>\n" +
				"### Nested Schema for `interfaces.security_groups`\n\n" +
				"Read-Only:\n\n" +
				"- `group_id` (String) Group ID.\n",
			want: map[string][]Attribute{
				"interfaces.security_groups": {
					{Name: "group_id", Description: "Group ID.", OriginalDescription: "Group ID.", Type: "string", ReadOnly: true},
				},
			},
		},
		{
			name: "normalizes nestedobjatt-- anchor prefix",
			content: "<a id=\"nestedobjatt--severity_summary\"></a>\n" +
				"### Nested Schema for `severity_summary`\n\n" +
				"Read-Only:\n\n" +
				"- `critical` (Number)\n" +
				"- `total` (Number)\n",
			want: map[string][]Attribute{
				"severity_summary": {
					{Name: "critical", Description: "", OriginalDescription: "", Type: "number", ReadOnly: true},
					{Name: "total", Description: "", OriginalDescription: "", Type: "number", ReadOnly: true},
				},
			},
		},
		{
			name: "parses required and optional sub-sections",
			content: "<a id=\"nestedatt--config\"></a>\n" +
				"### Nested Schema for `config`\n\n" +
				"Required:\n\n" +
				"- `key` (String) Key.\n\n" +
				"Optional:\n\n" +
				"- `value` (String) Value.\n",
			want: map[string][]Attribute{
				"config": {
					{Name: "key", Description: "Key.", OriginalDescription: "Key.", Type: "string", Required: true},
					{Name: "value", Description: "Value.", OriginalDescription: "Value.", Type: "string"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNestedSchemas(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAssignNestedAttrs(t *testing.T) {
	tests := []struct {
		name      string
		attrs     []Attribute
		nestedMap map[string][]Attribute
		prefix    string
		want      []Attribute
	}{
		{
			name: "assigns nested attributes and recurses deeply",
			attrs: []Attribute{
				{Name: "interfaces", Type: "array(object)"},
				{Name: "name", Type: "string"},
			},
			nestedMap: map[string][]Attribute{
				"interfaces": {
					{Name: "id", Type: "string"},
					{Name: "security_groups", Type: "array(object)"},
				},
				"interfaces.security_groups": {
					{Name: "group_id", Type: "string"},
				},
			},
			prefix: "",
			want: []Attribute{
				{
					Name: "interfaces", Type: "array(object)",
					NestedAttributes: []Attribute{
						{Name: "id", Type: "string"},
						{
							Name: "security_groups", Type: "array(object)",
							NestedAttributes: []Attribute{
								{Name: "group_id", Type: "string"},
							},
						},
					},
				},
				{Name: "name", Type: "string"},
			},
		},
		{
			name:      "no-op when nested map is empty",
			attrs:     []Attribute{{Name: "id", Type: "string"}},
			nestedMap: map[string][]Attribute{},
			prefix:    "",
			want:      []Attribute{{Name: "id", Type: "string"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignNestedAttrs(tt.attrs, tt.nestedMap, tt.prefix)
			if !reflect.DeepEqual(tt.attrs, tt.want) {
				t.Errorf("got %+v, want %+v", tt.attrs, tt.want)
			}
		})
	}
}

func TestParseSchema(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Attribute
	}{
		{
			name: "full schema with required, read-only, and nested attributes",
			content: "## Schema\n\n" +
				"### Required\n\n" +
				"- `id` (String) ID of machine-type.\n\n" +
				"### Read-Only\n\n" +
				"- `name` (String) Name of instance.\n" +
				"- `interfaces` (Attributes List) Network interfaces. (see [below for nested schema](#nestedatt--interfaces))\n\n" +
				"<a id=\"nestedatt--interfaces\"></a>\n" +
				"### Nested Schema for `interfaces`\n\n" +
				"Read-Only:\n\n" +
				"- `ip` (String) IP address.\n\n" +
				"## Import\n",
			want: []Attribute{
				{Name: "id", Description: "ID of machine-type.", OriginalDescription: "ID of machine-type.", Type: "string", Required: true},
				{Name: "name", Description: "Name of instance.", OriginalDescription: "Name of instance.", Type: "string", ReadOnly: true},
				{
					Name: "interfaces", Description: "Network interfaces.", OriginalDescription: "Network interfaces.", Type: "array(object)", ReadOnly: true,
					NestedAttributes: []Attribute{
						{Name: "ip", Description: "IP address.", OriginalDescription: "IP address.", Type: "string", ReadOnly: true},
					},
				},
			},
		},
		{
			name:    "no schema section returns empty slice",
			content: "# Title\n\nNo schema here.\n",
			want:    []Attribute{},
		},
		{
			name:    "schema with only optional attributes",
			content: "## Schema\n\n### Optional\n\n- `name` (String) Optional name.\n",
			want: []Attribute{
				{Name: "name", Description: "Optional name.", OriginalDescription: "Optional name.", Type: "string"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSchema(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestExtractCodeBlock(t *testing.T) {
	tests := []struct {
		name    string
		content string
		lang    string
		want    *string
	}{
		{
			name:    "extracts terraform block",
			content: "Some text\n```terraform\nresource \"mgc_vm\" \"ex\" {}\n```\nMore text",
			lang:    "terraform",
			want:    strPtr(`resource "mgc_vm" "ex" {}`),
		},
		{
			name:    "extracts shell block",
			content: "```shell\nterraform import mgc_vm.example <id>\n```",
			lang:    "shell",
			want:    strPtr("terraform import mgc_vm.example <id>"),
		},
		{
			name:    "returns nil when language block not found",
			content: "no code block here",
			lang:    "terraform",
			want:    nil,
		},
		{
			name:    "returns nil for unclosed block",
			content: "```terraform\nresource \"mgc_vm\" \"ex\" {}",
			lang:    "terraform",
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeBlock(tt.content, tt.lang)
			if (got == nil) != (tt.want == nil) {
				gotStr, wantStr := "<nil>", "<nil>"
				if got != nil {
					gotStr = *got
				}
				if tt.want != nil {
					wantStr = *tt.want
				}
				t.Errorf("got %q, want %q", gotStr, wantStr)
				return
			}
			if got != nil && *got != *tt.want {
				t.Errorf("got %q, want %q", *got, *tt.want)
			}
		})
	}
}

func TestParseImport(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "extracts import command from shell block",
			content: "## Schema\n\nSome schema.\n\n## Import\n\n```shell\nterraform import mgc_vm.example <id>\n```\n",
			want:    "terraform import mgc_vm.example <id>",
		},
		{
			name:    "returns empty when no import section",
			content: "## Schema\n\nsome content\n",
			want:    "",
		},
		{
			name:    "returns empty when import section has no shell block",
			content: "## Import\n\nNo code block here.\n",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseImport(tt.content)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeeplURL(t *testing.T) {
	tests := []struct {
		apiKey string
		want   string
	}{
		{"abc123:fx", "https://api-free.deepl.com/v2/translate"},
		{"abc123", "https://api.deepl.com/v2/translate"},
		{":fx", "https://api-free.deepl.com/v2/translate"},
		{"abc:fxyz", "https://api.deepl.com/v2/translate"},
	}
	for _, tt := range tests {
		t.Run(tt.apiKey, func(t *testing.T) {
			got := deeplURL(tt.apiKey)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrepareTranslations(t *testing.T) {
	tests := []struct {
		name        string
		output      map[string]Entry
		existing    map[string]Entry
		wantPending []string
		wantOutput  map[string]Entry
	}{
		{
			name: "new entry with no existing file adds non-empty originals to pending",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Name of instance", OriginalDescription: "Name of instance"},
					{Name: "id", Description: "", OriginalDescription: ""},
				}},
			},
			existing:    nil,
			wantPending: []string{"Name of instance"},
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Name of instance", OriginalDescription: "Name of instance"},
					{Name: "id", Description: "", OriginalDescription: ""},
				}},
			},
		},
		{
			name: "existing translation is reused when original text is unchanged",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Name of instance", OriginalDescription: "Name of instance"},
				}},
			},
			existing: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Nome da instância", OriginalDescription: "Name of instance"},
				}},
			},
			wantPending: nil,
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Nome da instância", OriginalDescription: "Name of instance"},
				}},
			},
		},
		{
			name: "changed original description triggers re-translation",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Updated description", OriginalDescription: "Updated description"},
				}},
			},
			existing: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Tradução antiga", OriginalDescription: "Old description"},
				}},
			},
			wantPending: []string{"Updated description"},
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Updated description", OriginalDescription: "Updated description"},
				}},
			},
		},
		{
			name: "identical descriptions across attributes are deduplicated",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "a", Description: "Same text", OriginalDescription: "Same text"},
					{Name: "b", Description: "Same text", OriginalDescription: "Same text"},
				}},
			},
			existing:    nil,
			wantPending: []string{"Same text"},
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "a", Description: "Same text", OriginalDescription: "Same text"},
					{Name: "b", Description: "Same text", OriginalDescription: "Same text"},
				}},
			},
		},
		{
			name: "existing translation is reused for nested attributes",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{
						Name: "interfaces", Description: "Network interfaces", OriginalDescription: "Network interfaces",
						NestedAttributes: []Attribute{
							{Name: "ip", Description: "IP address", OriginalDescription: "IP address"},
						},
					},
				}},
			},
			existing: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{
						Name: "interfaces", Description: "Interfaces de rede", OriginalDescription: "Network interfaces",
						NestedAttributes: []Attribute{
							{Name: "ip", Description: "Endereço IP", OriginalDescription: "IP address"},
						},
					},
				}},
			},
			wantPending: nil,
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{
						Name: "interfaces", Description: "Interfaces de rede", OriginalDescription: "Network interfaces",
						NestedAttributes: []Attribute{
							{Name: "ip", Description: "Endereço IP", OriginalDescription: "IP address"},
						},
					},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareTranslations(tt.output, tt.existing)
			if !reflect.DeepEqual(got, tt.wantPending) {
				t.Errorf("pending: got %v, want %v", got, tt.wantPending)
			}
			if !reflect.DeepEqual(tt.output, tt.wantOutput) {
				t.Errorf("output: got %+v, want %+v", tt.output, tt.wantOutput)
			}
		})
	}
}

func TestApplyTranslations(t *testing.T) {
	tests := []struct {
		name         string
		output       map[string]Entry
		translations map[string]string
		wantOutput   map[string]Entry
	}{
		{
			name: "applies translation to matching attribute",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Name of instance", OriginalDescription: "Name of instance"},
				}},
			},
			translations: map[string]string{"Name of instance": "Nome da instância"},
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Nome da instância", OriginalDescription: "Name of instance"},
				}},
			},
		},
		{
			name: "leaves description unchanged when no matching translation",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Original", OriginalDescription: "Original"},
				}},
			},
			translations: map[string]string{},
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{Name: "name", Description: "Original", OriginalDescription: "Original"},
				}},
			},
		},
		{
			name: "applies translation to nested attributes",
			output: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{
						Name: "interfaces", Description: "Network interfaces", OriginalDescription: "Network interfaces",
						NestedAttributes: []Attribute{
							{Name: "ip", Description: "IP address", OriginalDescription: "IP address"},
						},
					},
				}},
			},
			translations: map[string]string{
				"Network interfaces": "Interfaces de rede",
				"IP address":         "Endereço IP",
			},
			wantOutput: map[string]Entry{
				"resource.vm": {Attributes: []Attribute{
					{
						Name: "interfaces", Description: "Interfaces de rede", OriginalDescription: "Network interfaces",
						NestedAttributes: []Attribute{
							{Name: "ip", Description: "Endereço IP", OriginalDescription: "IP address"},
						},
					},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyTranslations(tt.output, tt.translations)
			if !reflect.DeepEqual(tt.output, tt.wantOutput) {
				t.Errorf("got %+v, want %+v", tt.output, tt.wantOutput)
			}
		})
	}
}

func TestLoadExistingFile(t *testing.T) {
	validPath := createTempJSONFile(t, `{"resource.vm": {"command": null, "attributes": []}}`)
	invalidPath := createTempJSONFile(t, `{invalid json}`)

	tests := []struct {
		name    string
		path    string
		wantNil bool
		wantKey string
	}{
		{"non-existent file", "/nonexistent/path/commands.json", true, ""},
		{"valid JSON file", validPath, false, "resource.vm"},
		{"invalid JSON file", invalidPath, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loadExistingFile(tt.path)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got non-nil map")
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if tt.wantKey != "" {
				if _, ok := got[tt.wantKey]; !ok {
					t.Errorf("expected key %q in result", tt.wantKey)
				}
			}
		})
	}
}
