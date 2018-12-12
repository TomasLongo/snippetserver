package snippet

import "bytes"

type Snippet struct {
	Source string
	variables map[string]string
}

func (s *Snippet) String() string {
	buffer := new(bytes.Buffer)
	buffer.WriteString("---\n")
	for k, v := range s.variables {
		buffer.WriteString(k)
		buffer.WriteString(": ")
		buffer.WriteString(v)
		buffer.WriteString("\n")
	}
	buffer.WriteString("---\n")
	buffer.WriteString(s.Source)
	buffer.WriteString("\n")

	return buffer.String()
}

func (s *Snippet)AddVar(key string, value string) {
	s.variables[key] = value
}

func (s *Snippet)GetVar(key string) string {
	return s.variables[key]
}

func NewSnippet() *Snippet {
	snippet := new(Snippet)
	snippet.variables = make(map[string]string)

	return snippet
}
