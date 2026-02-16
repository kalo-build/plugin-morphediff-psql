package formatdef

import (
	"fmt"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/kalo-build/go-util/strcase"
)

// Client for pluralization, initialized once
var pluralizeClient = pluralize.NewClient()

// ContentBuilder helps generate formatted code with proper indentation
type ContentBuilder struct {
	lines       []string
	indentLevel int
	indentStr   string
}

// NewContentBuilder creates a new content builder
func NewContentBuilder(indentStr string) *ContentBuilder {
	if indentStr == "" {
		indentStr = "    " // Default to 4 spaces
	}
	return &ContentBuilder{
		lines:     []string{},
		indentStr: indentStr,
	}
}

// Line adds a line with current indentation
func (b *ContentBuilder) Line(format string, args ...interface{}) *ContentBuilder {
	line := fmt.Sprintf(format, args...)
	if line == "" {
		b.lines = append(b.lines, "")
	} else {
		indent := strings.Repeat(b.indentStr, b.indentLevel)
		b.lines = append(b.lines, indent+line)
	}
	return b
}

// Indent increases indentation level
func (b *ContentBuilder) Indent() *ContentBuilder {
	b.indentLevel++
	return b
}

// Dedent decreases indentation level
func (b *ContentBuilder) Dedent() *ContentBuilder {
	if b.indentLevel > 0 {
		b.indentLevel--
	}
	return b
}

// Comment adds a comment line
func (b *ContentBuilder) Comment(format string, args ...interface{}) *ContentBuilder {
	return b.Line("-- "+format, args...)
}

// BlockComment adds a multi-line comment
func (b *ContentBuilder) BlockComment(lines ...string) *ContentBuilder {
	// TODO: Adjust for your format's block comment syntax
	b.Line("/*")
	for _, line := range lines {
		b.Line(" * %s", line)
	}
	b.Line(" */")
	return b
}

// AppendToLastLine appends text to the last line (useful for adding commas, etc)
func (b *ContentBuilder) AppendToLastLine(text string) *ContentBuilder {
	if len(b.lines) > 0 {
		b.lines[len(b.lines)-1] += text
	}
	return b
}

// Build returns the final content as a byte array
// Adds 2 trailing newlines to match expected output format
func (b *ContentBuilder) Build() []byte {
	content := strings.Join(b.lines, "\n")
	// Add 2 trailing newlines to match expected output format
	if content != "" {
		content += "\n\n"
	}
	return []byte(content)
}

// String returns the content as a string
func (b *ContentBuilder) String() string {
	return strings.Join(b.lines, "\n")
}

// Common formatting helpers

// FormatList formats a list of items with a separator
func FormatList(items []string, separator string) string {
	return strings.Join(items, separator)
}

// QuoteString adds quotes around a string (adjust for your format)
func QuoteString(s string) string {
	// TODO: Adjust quote style for your format (single vs double quotes)
	return fmt.Sprintf(`"%s"`, s)
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if len(pascal) > 0 {
		return strings.ToLower(pascal[:1]) + pascal[1:]
	}
	return pascal
}

// ToSnakeCase converts a string to snake_case using the strcase package
func ToSnakeCase(s string) string {
	return strcase.ToSnakeCaseLower(s)
}

// Pluralize pluralizes a word using English rules
func Pluralize(word string) string {
	return pluralizeClient.Plural(word)
}

// ToTableName converts a model name to a PostgreSQL table name (snake_case, pluralized)
// This matches the naming convention used by plugin-morphe-psql-types
func ToTableName(modelName string) string {
	return Pluralize(ToSnakeCase(modelName))
}

// ToForeignKeyConstraintName generates a FK constraint name matching plugin-morphe-psql-types
func ToForeignKeyConstraintName(tableName, relationshipName string) string {
	return fmt.Sprintf("fk_%s_%s", ToSnakeCase(tableName), ToSnakeCase(relationshipName))
}

// ToForeignKeyColumnName generates a FK column name
func ToForeignKeyColumnName(relationshipName string) string {
	return ToSnakeCase(relationshipName) + "_id"
}

// ToPolymorphicTypeColumnName generates a polymorphic type column name (e.g., "owner_type")
func ToPolymorphicTypeColumnName(relationshipName string) string {
	return ToSnakeCase(relationshipName) + "_type"
}

// ToPolymorphicIdColumnName generates a polymorphic ID column name (e.g., "owner_id")
func ToPolymorphicIdColumnName(relationshipName string) string {
	return ToSnakeCase(relationshipName) + "_id"
}

// ToPolymorphicIndexName generates an index name for polymorphic columns
func ToPolymorphicIndexName(tableName, relationshipName string) string {
	return fmt.Sprintf("idx_%s_%s", ToSnakeCase(tableName), ToSnakeCase(relationshipName))
}

// IsPolymorphicRelationType returns true if the relationship type is polymorphic
func IsPolymorphicRelationType(relationType string) bool {
	switch relationType {
	case "ForOnePoly", "ForManyPoly", "HasOnePoly", "HasManyPoly":
		return true
	default:
		return false
	}
}

// IsForwardPolymorphicType returns true if the relationship type stores the polymorphic columns
// (ForOnePoly, ForManyPoly store the type/id columns on the model)
func IsForwardPolymorphicType(relationType string) bool {
	switch relationType {
	case "ForOnePoly", "ForManyPoly":
		return true
	default:
		return false
	}
}
