package render

import (
	"html/template"
	"strings"

	"github.com/wamphlett/blogsync/pkg/blog"
)

const (
	columnsPerRow  = 3
	imageQueryArgs = "w=400&c=postThumb"
)

const tableTemplateSrc = `<table id="{{.ID}}">
{{- range .Rows}}
  <tr>
{{- range .}}
    <td width="33%" valign="top">
      <img src="{{.ImageURL}}" width="100%" alt="{{.Title}}" /><br />
      <a href="{{.URL}}"><b>{{.Title}}</b></a><br />
      {{.Description}}
    </td>
{{- end}}
  </tr>
{{- end}}
</table>`

// article is the view model fed to the table template.
type article struct {
	Title       string
	Description string
	ImageURL    string
	URL         string
}

var tableTemplate = template.Must(template.New("table").Parse(tableTemplateSrc))

// Table builds the full <table id="..."> HTML block for the given articles,
// wrapping them into rows of columnsPerRow columns each.
func Table(id, baseURL string, articles []blog.Article) (string, error) {
	items := make([]article, 0, len(articles))
	for _, a := range articles {
		items = append(items, article{
			Title:       a.Title,
			Description: a.Description,
			ImageURL:    a.Image + "?" + imageQueryArgs,
			URL:         strings.TrimRight(baseURL, "/") + "/" + a.TopicSlug + "/" + a.Slug,
		})
	}

	var rows [][]article
	for i := 0; i < len(items); i += columnsPerRow {
		end := i + columnsPerRow
		if end > len(items) {
			end = len(items)
		}
		rows = append(rows, items[i:end])
	}

	data := struct {
		ID   string
		Rows [][]article
	}{ID: id, Rows: rows}

	var buf strings.Builder
	if err := tableTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
