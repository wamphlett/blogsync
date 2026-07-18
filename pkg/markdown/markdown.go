package markdown

import (
	"fmt"
	"regexp"
)

// ReplaceTable finds a `<table id="id">...</table>` block within content and
// replaces it wholesale with newTable. It returns an error if no matching
// table is found.
func ReplaceTable(content, id, newTable string) (string, error) {
	pattern := fmt.Sprintf(`(?s)<table id="%s">.*?</table>`, regexp.QuoteMeta(id))
	re := regexp.MustCompile(pattern)

	if !re.MatchString(content) {
		return "", fmt.Errorf(`no <table id=%q> found in markdown file`, id)
	}

	return re.ReplaceAllLiteralString(content, newTable), nil
}
