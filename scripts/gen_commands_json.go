package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

type Attribute struct {
	Name                string      `json:"name"`
	Description         string      `json:"description"`
	OriginalDescription string      `json:"original_description"`
	Type                string      `json:"type"`
	Required            bool        `json:"required"`
	ReadOnly            bool        `json:"read_only"`
	DefaultValue        any         `json:"default_value"`
	NestedAttributes    []Attribute `json:"nested_attributes,omitempty"`
}

type Entry struct {
	Command    *string     `json:"command"`
	Attributes []Attribute `json:"attributes"`
	Import     string      `json:"import,omitempty"`
}

type partialAttr struct {
	name     string
	typeStr  string
	required bool
	readOnly bool
	desc     string
}

type deeplRequest struct {
	Text       []string `json:"text"`
	TargetLang string   `json:"target_lang"`
}

type deeplResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
}

const deeplBatchSize = 50

var typeMap = map[string]string{
	"String":          "string",
	"Number":          "number",
	"Boolean":         "bool",
	"List of String":  "array(string)",
	"List of Number":  "array(number)",
	"Set of String":   "set(string)",
	"Set of Number":   "set(number)",
	"Map of String":   "map(string)",
	"Map of Number":   "map(number)",
	"Attributes List": "array(object)",
	"Attributes":      "object",
	"Attributes Set":  "set(object)",
	"Map":             "map",
}

func parseType(typeRaw string) string {
	base := strings.TrimSpace(strings.SplitN(typeRaw, ",", 2)[0])
	if mapped, ok := typeMap[base]; ok {
		return mapped
	}
	return strings.ToLower(base)
}

func extractSection(content, header string, stops []string) string {
	marker := header + "\n"
	idx := strings.Index(content, marker)
	if idx == -1 {
		return ""
	}
	rest := content[idx+len(marker):]
	end := len(rest)
	for _, stop := range stops {
		if i := strings.Index(rest, stop); i != -1 && i < end {
			end = i
		}
	}
	return rest[:end]
}

var (
	reSeeBelow   = regexp.MustCompile(`\s*\(see \[below for nested schema\]\([^)]*\)\)`)
	reWhitespace = regexp.MustCompile(`\s+`)
	reNestedPath = regexp.MustCompile("### Nested Schema for `([^`]+)`")
)

func removeSeeBelow(s string) string {
	return strings.TrimSpace(reSeeBelow.ReplaceAllString(s, ""))
}

// Returns (name, typeStr, description, ok).
func parseAttrLine(line string) (string, string, string, bool) {
	if !strings.HasPrefix(line, "- `") {
		return "", "", "", false
	}

	nameStart := 3
	nameEnd := strings.Index(line[nameStart:], "`")
	if nameEnd == -1 {
		return "", "", "", false
	}

	name := line[nameStart : nameStart+nameEnd]

	after := line[nameStart+nameEnd+1:]
	if !strings.HasPrefix(after, " (") {
		return "", "", "", false
	}

	rest := after[2:]
	depth, i := 1, 0
	for i < len(rest) && depth > 0 {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		i++
	}
	if depth != 0 {
		return "", "", "", false
	}
	typeStr := rest[:i-1]

	desc := strings.TrimSpace(rest[i:])

	return name, typeStr, desc, true
}

func (p *partialAttr) finalize() Attribute {
	desc := strings.TrimSpace(reWhitespace.ReplaceAllString(p.desc, " "))
	return Attribute{
		Name:                p.name,
		Description:         desc,
		OriginalDescription: desc,
		Type:                parseType(p.typeStr),
		Required:            p.required,
		ReadOnly:            p.readOnly,
		DefaultValue:        nil,
	}
}

func parseAttrSection(content string, required bool, readOnly bool) []Attribute {
	var attrs []Attribute
	var newAttr *partialAttr

	for line := range strings.SplitSeq(content, "\n") {
		if name, typeStr, desc, ok := parseAttrLine(line); ok {
			if newAttr != nil {
				attrs = append(attrs, newAttr.finalize())
			}
			newAttr = &partialAttr{
				name:     name,
				typeStr:  typeStr,
				required: required,
				readOnly: readOnly,
				desc:     removeSeeBelow(desc),
			}
		} else if newAttr != nil {
			s := strings.TrimSpace(line)
			if s == "" {
				continue
			}
			switch s[0] {
			case '#', '>', '|', '<', '*':
				continue
			}
			if s == "Read-Only:" || s == "Optional:" || s == "Required:" {
				continue
			}
			s = removeSeeBelow(s)
			if s != "" {
				newAttr.desc += " " + s
			}
		}
	}
	if newAttr != nil {
		attrs = append(attrs, newAttr.finalize())
	}
	return attrs
}

func parseNestedSchemas(content string) map[string][]Attribute {
	nestedMap := make(map[string][]Attribute)

	// Normalize alternate anchor prefix so both variants are handled uniformly.
	content = strings.ReplaceAll(content, `<a id="nestedobjatt--`, `<a id="nestedatt--`)

	const anchorPrefix = `<a id="nestedatt--`
	parts := strings.Split(content, anchorPrefix)

	for _, part := range parts[1:] {
		if !strings.Contains(part, `"></a>`) {
			continue
		}
		m := reNestedPath.FindStringSubmatch(part)
		if m == nil {
			continue
		}
		attrPath := m[1]
		headingEnd := strings.Index(part, m[0]) + len(m[0])
		sectionContent := part[headingEnd:]
		if i := strings.Index(sectionContent, "\n## "); i != -1 {
			sectionContent = sectionContent[:i]
		}

		var attrs []Attribute
		for _, sub := range []struct {
			header   string
			stops    []string
			required bool
			readOnly bool
		}{
			{"Required:", []string{"\nOptional:", "\nRead-Only:"}, true, false},
			{"Optional:", []string{"\nRequired:", "\nRead-Only:"}, false, false},
			{"Read-Only:", []string{"\nRequired:", "\nOptional:"}, false, true},
		} {
			if sec := extractSection(sectionContent, sub.header, sub.stops); sec != "" {
				attrs = append(attrs, parseAttrSection(sec, sub.required, sub.readOnly)...)
			}
		}
		nestedMap[attrPath] = attrs
	}
	return nestedMap
}

func assignNestedAttrs(attrs []Attribute, nestedMap map[string][]Attribute, prefix string) {
	for i := range attrs {
		path := attrs[i].Name
		if prefix != "" {
			path = prefix + "." + attrs[i].Name
		}
		if nested, ok := nestedMap[path]; ok {
			attrs[i].NestedAttributes = nested
			assignNestedAttrs(attrs[i].NestedAttributes, nestedMap, path)
		}
	}
}

func parseSchema(content string) []Attribute {
	schemaSection := extractSection(content, "## Schema", []string{"\n## "})
	if schemaSection == "" {
		return []Attribute{}
	}

	stops := []string{"\n### ", "\n<a id"}

	var attrs []Attribute

	for _, s := range []struct {
		header   string
		required bool
		readOnly bool
	}{
		{"### Required", true, false},
		{"### Optional", false, false},
		{"### Read-Only", false, true},
	} {
		if sec := extractSection(schemaSection, s.header, stops); sec != "" {
			attrs = append(attrs, parseAttrSection(sec, s.required, s.readOnly)...)
		}
	}

	nestedMap := parseNestedSchemas(content)
	assignNestedAttrs(attrs, nestedMap, "")

	if attrs == nil {
		return []Attribute{}
	}
	return attrs
}

func extractCodeBlock(content, lang string) *string {
	marker := "```" + lang + "\n"
	start := strings.Index(content, marker)
	if start == -1 {
		return nil
	}
	start += len(marker)
	end := strings.Index(content[start:], "```")
	if end == -1 {
		return nil
	}
	result := strings.TrimSpace(content[start : start+end])
	return &result
}

func parseExample(content string) *string {
	return extractCodeBlock(content, "terraform")
}

func parseImport(content string) string {
	importSection := extractSection(content, "## Import", []string{"\n## "})
	if importSection == "" {
		return ""
	}
	if cmd := extractCodeBlock(importSection, "shell"); cmd != nil {
		return *cmd
	}
	return ""
}

func sortedMDFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func processDir(dir, prefix string, output map[string]Entry) error {
	files, err := sortedMDFiles(dir)
	if err != nil {
		return err
	}
	for _, filename := range files {
		name := strings.TrimSuffix(filename, ".md")
		raw, err := os.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			continue
		}
		content := string(raw)
		entry := Entry{
			Command:    parseExample(content),
			Attributes: parseSchema(content),
		}
		if imp := parseImport(content); imp != "" {
			entry.Import = imp
		}
		output[prefix+name] = entry
	}
	return nil
}

func deeplURL(apiKey string) string {
	if strings.HasSuffix(apiKey, ":fx") {
		return "https://api-free.deepl.com/v2/translate"
	}
	return "https://api.deepl.com/v2/translate"
}

func translateChunk(texts []string, apiKey string) ([]string, error) {
	body, err := json.Marshal(deeplRequest{Text: texts, TargetLang: "PT-BR"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, deeplURL(apiKey), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "DeepL-Auth-Key "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepl API returned status %d: %s", resp.StatusCode, bytes.TrimSpace(errBody))
	}

	var result deeplResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	translated := make([]string, len(result.Translations))
	for i, tr := range result.Translations {
		translated[i] = tr.Text
	}
	return translated, nil
}

func loadExistingFile(path string) map[string]Entry {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out map[string]Entry
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func prepareTranslations(output map[string]Entry, existing map[string]Entry) []string {
	seen := make(map[string]bool)
	var pending []string

	var visitAttrs func(newAttrs []Attribute, existingAttrs []Attribute)
	visitAttrs = func(newAttrs []Attribute, existingAttrs []Attribute) {
		byName := make(map[string]Attribute, len(existingAttrs))
		for _, attr := range existingAttrs {
			byName[attr.Name] = attr
		}
		for i := range newAttrs {
			attr := byName[newAttrs[i].Name]
			orig := newAttrs[i].OriginalDescription

			if attr.OriginalDescription == orig {
				newAttrs[i].Description = attr.Description
			} else if orig != "" && !seen[orig] {
				seen[orig] = true
				pending = append(pending, orig)
			}
			visitAttrs(newAttrs[i].NestedAttributes, attr.NestedAttributes)
		}
	}

	for key := range output {
		entry := output[key]
		var existingAttrs []Attribute
		if ee, ok := existing[key]; ok {
			existingAttrs = ee.Attributes
		}
		visitAttrs(entry.Attributes, existingAttrs)
		output[key] = entry
	}
	return pending
}

func translatePending(pending []string, translations map[string]string, apiKey string) error {
	fmt.Printf("Translating %d descriptions\n", len(pending))

	for i := 0; i < len(pending); i += deeplBatchSize {
		end := min(i+deeplBatchSize, len(pending))
		batch := pending[i:end]
		results, err := translateChunk(batch, apiKey)
		if err != nil {
			return fmt.Errorf("batch %d: %w", i/deeplBatchSize+1, err)
		}
		for j, orig := range batch {
			translations[orig] = results[j]
		}
	}
	return nil
}

func applyTranslations(output map[string]Entry, translations map[string]string) {
	var visitAttrs func([]Attribute) []Attribute
	visitAttrs = func(attrs []Attribute) []Attribute {
		for i := range attrs {
			if t, ok := translations[attrs[i].OriginalDescription]; ok {
				attrs[i].Description = t
			}
			attrs[i].NestedAttributes = visitAttrs(attrs[i].NestedAttributes)
		}
		return attrs
	}
	for key := range output {
		entry := output[key]
		entry.Attributes = visitAttrs(entry.Attributes)
		output[key] = entry
	}
}

func writeJSON(f *os.File, v any) error {
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func writeFile(path string, v any) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "commands-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := writeJSON(tmp, v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	return nil
}

func main() {
	baseDir := "../"
	output := make(map[string]Entry)

	if err := processDir(filepath.Join(baseDir, "docs/resources"), "resource.", output); err != nil {
		fmt.Fprintf(os.Stderr, "error reading resources: %v\n", err)
		os.Exit(1)
	}
	if err := processDir(filepath.Join(baseDir, "docs/data-sources"), "datasource.", output); err != nil {
		fmt.Fprintf(os.Stderr, "error reading data-sources: %v\n", err)
		os.Exit(1)
	}

	outputPath := filepath.Join(baseDir, "commands.json")

	godotenv.Load("../.env")
	apiKey := os.Getenv("DEEPL_API_KEY")

	existing := loadExistingFile(outputPath)
	pending := prepareTranslations(output, existing)

	if apiKey == "" {
		fmt.Println("DEEPL_API_KEY not set, skipping translation")
	} else if len(pending) > 0 {
		translations := make(map[string]string)
		if err := translatePending(pending, translations, apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "translation error: %v — descriptions will fallback to original\n", err)
		}
		applyTranslations(output, translations)
	} else {
		fmt.Println("All descriptions up to date, skipping DeepL call")
	}

	if err := writeFile(outputPath, output); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
}
