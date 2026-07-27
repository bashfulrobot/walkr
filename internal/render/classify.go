package render

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// classifyLine wraps tokens in a code line with the same span classes
// prototype/assets/style.css expects (.kw/.tp/.st/.nm/.pn/.cm). This is a
// small heuristic classifier for the languages walkr itself teaches
// with, not a general tokenizer — see docs/ai/content-format.md.
func classifyLine(lang, line string) string {
	switch lang {
	case "go", "golang":
		return classifyGoLine(line)
	case "yaml", "yml":
		return classifyYAMLLine(line)
	default:
		return html.EscapeString(line)
	}
}

var goKeywords = map[string]bool{
	"func": true, "return": true, "if": true, "else": true, "for": true,
	"range": true, "var": true, "const": true, "type": true, "struct": true,
	"interface": true, "package": true, "import": true, "switch": true,
	"case": true, "default": true, "break": true, "continue": true,
	"defer": true, "go": true, "chan": true, "select": true, "map": true,
	"nil": true, "true": true, "false": true,
}

var goBuiltinTypes = map[string]bool{
	"string": true, "error": true, "int": true, "bool": true, "byte": true,
	"rune": true, "float64": true, "float32": true, "any": true, "int64": true,
}

var goTokenRe = regexp.MustCompile(`//[^\n]*|"(?:[^"\\]|\\.)*"|\b\d+\b|\*?\b[A-Za-z_][A-Za-z0-9_]*\b`)

func classifyGoLine(line string) string {
	var out strings.Builder
	last := 0
	for _, loc := range goTokenRe.FindAllStringIndex(line, -1) {
		start, end := loc[0], loc[1]
		if start > last {
			out.WriteString(html.EscapeString(line[last:start]))
		}
		out.WriteString(classifyGoToken(line[start:end]))
		last = end
	}
	if last < len(line) {
		out.WriteString(html.EscapeString(line[last:]))
	}
	return out.String()
}

func classifyGoToken(tok string) string {
	if strings.HasPrefix(tok, "//") {
		return `<span class="cm">` + html.EscapeString(tok) + `</span>`
	}
	if strings.HasPrefix(tok, `"`) {
		return `<span class="st">` + html.EscapeString(tok) + `</span>`
	}
	prefix, bare := "", tok
	if strings.HasPrefix(tok, "*") {
		prefix, bare = "*", tok[1:]
	}
	if _, err := strconv.Atoi(bare); err == nil {
		return prefix + `<span class="nm">` + html.EscapeString(bare) + `</span>`
	}
	if goKeywords[bare] {
		return prefix + `<span class="kw">` + html.EscapeString(bare) + `</span>`
	}
	if goBuiltinTypes[bare] || (bare != "" && bare[0] >= 'A' && bare[0] <= 'Z') {
		return prefix + `<span class="tp">` + html.EscapeString(bare) + `</span>`
	}
	return prefix + html.EscapeString(bare)
}

var yamlKeyRe = regexp.MustCompile(`^(\s*(?:- )?)([A-Za-z_][\w.-]*)(:)(.*)$`)
var yamlInlineTokenRe = regexp.MustCompile(`[{}:,]|"[^"]*"|\b\d+\b|[\w./-]+`)

func classifyYAMLLine(line string) string {
	m := yamlKeyRe.FindStringSubmatch(line)
	if m == nil {
		return classifyYAMLValue(line)
	}
	indent, key, colon, rest := m[1], m[2], m[3], m[4]
	return html.EscapeString(indent) +
		`<span class="kw">` + html.EscapeString(key) + `</span>` +
		html.EscapeString(colon) +
		classifyYAMLValue(rest)
}

// classifyYAMLValue handles the scalar (or inline-mapping) after a `key:`.
func classifyYAMLValue(raw string) string {
	trimmed := strings.TrimLeft(raw, " ")
	leading := raw[:len(raw)-len(trimmed)]
	if trimmed == "" {
		return html.EscapeString(raw)
	}
	if strings.Contains(trimmed, "{") {
		return html.EscapeString(leading) + classifyYAMLInline(trimmed)
	}
	if _, err := strconv.Atoi(trimmed); err == nil {
		return html.EscapeString(leading) + `<span class="nm">` + html.EscapeString(trimmed) + `</span>`
	}
	return html.EscapeString(leading) + `<span class="st">` + html.EscapeString(trimmed) + `</span>`
}

// classifyYAMLInline handles `{ path: /healthz, port: 8080 }`-style scalars.
func classifyYAMLInline(s string) string {
	var out strings.Builder
	last := 0
	expectKey := true
	for _, loc := range yamlInlineTokenRe.FindAllStringIndex(s, -1) {
		start, end := loc[0], loc[1]
		if start > last {
			out.WriteString(html.EscapeString(s[last:start]))
		}
		tok := s[start:end]
		switch tok {
		case "{", "}", ":", ",":
			out.WriteString(`<span class="pn">` + html.EscapeString(tok) + `</span>`)
			if tok == ":" {
				expectKey = false
			}
			if tok == "," {
				expectKey = true
			}
		default:
			if _, err := strconv.Atoi(tok); err == nil {
				out.WriteString(`<span class="nm">` + html.EscapeString(tok) + `</span>`)
			} else if expectKey {
				out.WriteString(`<span class="kw">` + html.EscapeString(tok) + `</span>`)
				expectKey = false
			} else {
				out.WriteString(`<span class="st">` + html.EscapeString(tok) + `</span>`)
			}
		}
		last = end
	}
	if last < len(s) {
		out.WriteString(html.EscapeString(s[last:]))
	}
	return out.String()
}
