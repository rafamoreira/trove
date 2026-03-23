package publish

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"github.com/rafamoreira/trove/internal/vault"
)

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head><title>Index</title></head>
<body>
<h1>Index</h1>
{{range .Groups}}
<h2>{{.Language}}</h2>
<ul>
{{range .Snippets}}<li><a href="{{.Language}}/{{.Name}}.html">{{.Name}}</a>{{if .Description}} — {{.Description}}{{end}}</li>
{{end}}</ul>
{{end}}
</body>
</html>
`))

var snippetTmpl = template.Must(template.New("snippet").Parse(`<!DOCTYPE html>
<html>
<head><title>{{.ID}}</title></head>
<body>
<p><a href="../index.html">← back</a></p>
<h1>{{.ID}}</h1>
<dl>
<dt>Language</dt><dd>{{.Language}}</dd>
{{if .Description}}<dt>Description</dt><dd>{{.Description}}</dd>{{end}}
{{if .Tags}}<dt>Tags</dt><dd>{{.Tags}}</dd>{{end}}
</dl>
<pre>{{.Body}}</pre>
</body>
</html>
`))

type langGroup struct {
	Language string
	Snippets []*vault.Snippet
}

type snippetPage struct {
	ID          string
	Language    string
	Description string
	Tags        string
	Body        string
}

func Generate(snippets []*vault.Snippet, outputDir string) (map[string]any, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	// Group by language.
	groupMap := make(map[string][]*vault.Snippet)
	for _, s := range snippets {
		groupMap[s.Language] = append(groupMap[s.Language], s)
	}
	var groups []langGroup
	for lang, items := range groupMap {
		groups = append(groups, langGroup{Language: lang, Snippets: items})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Language < groups[j].Language
	})

	// Write index.
	indexFile, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return nil, err
	}
	defer indexFile.Close()
	if err := indexTmpl.Execute(indexFile, struct{ Groups []langGroup }{groups}); err != nil {
		return nil, err
	}

	// Write snippet pages.
	pageCount := 1 // index
	for _, s := range snippets {
		langDir := filepath.Join(outputDir, s.Language)
		if err := os.MkdirAll(langDir, 0o755); err != nil {
			return nil, err
		}

		body, err := s.Body()
		if err != nil {
			return nil, err
		}

		tags := ""
		if len(s.Tags) > 0 {
			for i, tag := range s.Tags {
				if i > 0 {
					tags += ", "
				}
				tags += tag
			}
		}

		f, err := os.Create(filepath.Join(langDir, s.Name+".html"))
		if err != nil {
			return nil, err
		}
		err = snippetTmpl.Execute(f, snippetPage{
			ID:          s.ID,
			Language:    s.Language,
			Description: s.Description,
			Tags:        tags,
			Body:        body,
		})
		f.Close()
		if err != nil {
			return nil, err
		}
		pageCount++
	}

	return map[string]any{
		"output_dir":    outputDir,
		"snippet_count": len(snippets),
		"page_count":    pageCount,
	}, nil
}
