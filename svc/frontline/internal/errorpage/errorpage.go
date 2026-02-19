package errorpage

import (
	"bytes"
	_ "embed"
	"html/template"
	"sync"
)

//go:embed error.html
var defaultTemplate string

// Data contains all the fields available to the error page template.
type Data struct {
	StatusCode int
	Title      string
	Message    string
	ErrorCode  string
	DocsURL    string
	RequestID  string
}

// Renderer renders error pages from Data.
type Renderer interface {
	Render(data Data) ([]byte, error)
}

// defaultRenderer uses the embedded HTML template.
type defaultRenderer struct {
	once sync.Once
	tmpl *template.Template
	err  error
}

func (r *defaultRenderer) Render(data Data) ([]byte, error) {
	r.once.Do(func() {
		r.tmpl, r.err = template.New("error").Parse(defaultTemplate)
	})
	if r.err != nil {
		return nil, r.err
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// NewRenderer returns a Renderer that uses the default embedded error page template.
func NewRenderer() Renderer {
	//nolint:exhaustruct
	return &defaultRenderer{}
}
