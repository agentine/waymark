package waymark

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// patternSegment represents one segment of a parsed path template.
type patternSegment struct {
	literal  string // literal text (empty if variable)
	variable string // variable name (empty if literal)
	regex    string // regex constraint for the variable
	greedy   bool   // greedy match (for path prefix patterns)
}

// compiledPattern is a compiled path template ready for matching.
type compiledPattern struct {
	template string         // original template string
	regex    *regexp.Regexp  // compiled regex
	vars     []string        // ordered variable names
	isPrefix bool            // true if this is a prefix pattern
	reverse  []patternSegment // segments for URL building
}

// patternCache caches compiled patterns by template string.
var patternCache sync.Map

// compilePattern compiles a path template into a compiledPattern.
// Templates use {varname} and {varname:regex} syntax.
// If prefix is true, the pattern matches path prefixes rather than full paths.
func compilePattern(template string, prefix bool) (*compiledPattern, error) {
	key := template
	if prefix {
		key = "prefix:" + template
	}
	if cached, ok := patternCache.Load(key); ok {
		return cached.(*compiledPattern), nil
	}

	segments, err := parseTemplate(template)
	if err != nil {
		return nil, err
	}

	var regexBuf strings.Builder
	regexBuf.WriteString("^")

	var vars []string
	for _, seg := range segments {
		if seg.variable != "" {
			vars = append(vars, seg.variable)
			constraint := seg.regex
			if constraint == "" {
				constraint = "[^/]+"
			}
			regexBuf.WriteString("(")
			regexBuf.WriteString(constraint)
			regexBuf.WriteString(")")
		} else {
			regexBuf.WriteString(regexp.QuoteMeta(seg.literal))
		}
	}

	if !prefix {
		regexBuf.WriteString("$")
	}

	compiled, err := regexp.Compile(regexBuf.String())
	if err != nil {
		return nil, fmt.Errorf("waymark: invalid pattern %q: %w", template, err)
	}

	cp := &compiledPattern{
		template: template,
		regex:    compiled,
		vars:     vars,
		isPrefix: prefix,
		reverse:  segments,
	}

	patternCache.Store(key, cp)
	return cp, nil
}

// match tests path against the compiled pattern and returns extracted variables.
func (cp *compiledPattern) match(path string) (map[string]string, bool) {
	matches := cp.regex.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	vars := make(map[string]string, len(cp.vars))
	for i, name := range cp.vars {
		vars[name] = matches[i+1]
	}
	return vars, true
}

// matchedPrefix returns the portion of path that was matched by a prefix pattern.
func (cp *compiledPattern) matchedPrefix(path string) (string, map[string]string, bool) {
	loc := cp.regex.FindStringSubmatchIndex(path)
	if loc == nil {
		return "", nil, false
	}

	vars := make(map[string]string, len(cp.vars))
	matches := cp.regex.FindStringSubmatch(path)
	for i, name := range cp.vars {
		vars[name] = matches[i+1]
	}
	return path[:loc[1]], vars, true
}

// buildPath reconstructs a path from the pattern template and variable values.
func (cp *compiledPattern) buildPath(pairs map[string]string) (string, error) {
	var buf strings.Builder
	for _, seg := range cp.reverse {
		if seg.variable != "" {
			val, ok := pairs[seg.variable]
			if !ok {
				return "", fmt.Errorf("waymark: missing route variable %q", seg.variable)
			}
			buf.WriteString(val)
		} else {
			buf.WriteString(seg.literal)
		}
	}
	return buf.String(), nil
}

// parseTemplate splits a path template into segments of literals and variables.
func parseTemplate(template string) ([]patternSegment, error) {
	var segments []patternSegment
	s := template

	for len(s) > 0 {
		// Find the next variable start.
		idx := strings.IndexByte(s, '{')
		if idx < 0 {
			// Rest is literal.
			segments = append(segments, patternSegment{literal: s})
			break
		}

		// Add literal prefix.
		if idx > 0 {
			segments = append(segments, patternSegment{literal: s[:idx]})
		}

		// Find the closing brace.
		s = s[idx+1:]
		end := findClosingBrace(s)
		if end < 0 {
			return nil, fmt.Errorf("waymark: unclosed brace in pattern %q", template)
		}

		content := s[:end]
		s = s[end+1:]

		// Split variable name and optional regex constraint.
		varName, constraint, _ := strings.Cut(content, ":")
		if varName == "" {
			return nil, fmt.Errorf("waymark: empty variable name in pattern %q", template)
		}

		segments = append(segments, patternSegment{
			variable: varName,
			regex:    constraint,
		})
	}

	return segments, nil
}

// findClosingBrace finds the matching closing brace, respecting nested braces
// in regex constraints like {id:[0-9]{3}}.
func findClosingBrace(s string) int {
	depth := 0
	for i, c := range s {
		switch c {
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}
