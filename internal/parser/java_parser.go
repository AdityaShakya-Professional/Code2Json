package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// ParseJavaDTO parses a Java DTO class and returns a map with default values.
// It handles: primitive types, boxed types, String, collections, nested classes in same file.
func ParseJavaDTO(src string) (interface{}, error) {
	src = stripJavaComments(src)

	// Collect all class names defined in this file
	localClasses := collectJavaClassNames(src)

	// Find the primary (first public or first) class
	classes := extractJavaClasses(src)
	if len(classes) == 0 {
		return nil, fmt.Errorf("no Java class found in file")
	}

	// Two-pass: register empty maps first, then fill, so cross-references resolve.
	classMap := map[string]map[string]interface{}{}
	for _, cls := range classes {
		classMap[cls.name] = map[string]interface{}{}
	}
	for _, cls := range classes {
		filled := buildJavaFieldMap(cls.fields, localClasses, classMap)
		for k, v := range filled {
			classMap[cls.name][k] = v
		}
	}

	// Return primary class (first one)
	primary := classes[0]
	result := classMap[primary.name]
	if result == nil {
		return nil, fmt.Errorf("failed to parse class fields")
	}
	return result, nil
}

type javaClass struct {
	name   string
	fields []javaField
}

type javaField struct {
	typeName string
	name     string
}

var javaClassRe = regexp.MustCompile(`(?:public\s+|private\s+|protected\s+)?(?:static\s+)?class\s+(\w+)(?:\s+extends\s+\w+)?(?:\s+implements\s+[\w,\s]+)?\s*\{`)
var javaFieldRe = regexp.MustCompile(`(?:private|protected|public)\s+([\w<>,\s\[\]]+?)\s+(\w+)\s*;`)
var javaAnnotationRe = regexp.MustCompile(`@\w+(?:\([^)]*\))?\s*`)

func stripJavaComments(src string) string {
	// Remove /* */ comments
	blockComment := regexp.MustCompile(`(?s)/\*.*?\*/`)
	src = blockComment.ReplaceAllString(src, "")
	// Remove // comments
	lineComment := regexp.MustCompile(`//[^\n]*`)
	src = lineComment.ReplaceAllString(src, "")
	return src
}

func collectJavaClassNames(src string) map[string]bool {
	names := map[string]bool{}
	for _, m := range javaClassRe.FindAllStringSubmatch(src, -1) {
		names[m[1]] = true
	}
	return names
}

func extractJavaClasses(src string) []javaClass {
	var classes []javaClass
	matches := javaClassRe.FindAllStringSubmatchIndex(src, -1)

	for i, match := range matches {
		name := src[match[2]:match[3]]
		start := match[1] // position of '{'

		// Find matching closing brace
		end := findMatchingBrace(src, start-1)
		if end == -1 {
			continue
		}

		var nextStart int
		if i+1 < len(matches) {
			nextStart = matches[i+1][0]
		} else {
			nextStart = len(src)
		}
		_ = nextStart

		body := src[start:end]
		fields := extractJavaFields(body)
		classes = append(classes, javaClass{name: name, fields: fields})
	}
	return classes
}

func extractJavaFields(body string) []javaField {
	// Remove annotations first
	cleaned := javaAnnotationRe.ReplaceAllString(body, "")
	var fields []javaField
	for _, m := range javaFieldRe.FindAllStringSubmatch(cleaned, -1) {
		typeName := strings.TrimSpace(m[1])
		fieldName := strings.TrimSpace(m[2])
		fields = append(fields, javaField{typeName: typeName, name: fieldName})
	}
	return fields
}

func findMatchingBrace(src string, openPos int) int {
	depth := 0
	for i := openPos; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func buildJavaFieldMap(fields []javaField, localClasses map[string]bool, classMap map[string]map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for _, f := range fields {
		result[f.name] = javaDefaultValue(f.typeName, localClasses, classMap)
	}
	return result
}

func javaDefaultValue(typeName string, local map[string]bool, classMap map[string]map[string]interface{}) interface{} {
	t := strings.TrimSpace(typeName)

	// Handle arrays
	if strings.HasSuffix(t, "[]") {
		inner := strings.TrimSuffix(t, "[]")
		return []interface{}{javaDefaultValue(inner, local, classMap)}
	}

	// Handle generics: List<X>, Set<X>, Collection<X>
	if m := regexp.MustCompile(`(?:List|Set|Collection|ArrayList|LinkedList)<(.+)>`).FindStringSubmatch(t); m != nil {
		inner := strings.TrimSpace(m[1])
		return []interface{}{javaDefaultValue(inner, local, classMap)}
	}
	// Map<K,V>
	if m := regexp.MustCompile(`(?:Map|HashMap|LinkedHashMap)<(.+),(.+)>`).FindStringSubmatch(t); m != nil {
		return map[string]interface{}{"key": javaDefaultValue(strings.TrimSpace(m[1]), local, classMap)}
	}

	// Primitives & common types
	switch t {
	case "int", "Integer", "short", "Short", "byte", "Byte":
		return 0
	case "long", "Long":
		return 0
	case "float", "Float", "double", "Double":
		return 0.0
	case "boolean", "Boolean":
		return false
	case "char", "Character":
		return ""
	case "String":
		return ""
	case "BigDecimal", "BigInteger":
		return 0
	case "Date", "LocalDate", "LocalDateTime", "Instant", "ZonedDateTime":
		return "1970-01-01T00:00:00Z"
	case "UUID":
		return "00000000-0000-0000-0000-000000000000"
	case "Object":
		return nil
	case "void", "Void":
		return nil
	}

	// If it's a locally defined class, recurse (if already resolved, use it)
	if local[t] {
		if resolved, ok := classMap[t]; ok {
			return resolved
		}
		// Placeholder to avoid infinite recursion
		return map[string]interface{}{}
	}

	// Unknown external dependency — ignore (return nil, filtered out)
	return nil
}
