package filters

import (
	"snippetserver/snippet"
	"strings"
)

// SnippetFilter is a function that applies a condition on a snippet
type SnippetFilter func(snippet *snippet.Snippet) bool

// LanguageFilter creates a filter that filters a snippet based on its `language` property
func LanguageFilter(language string) SnippetFilter {
	return func(snippet *snippet.Snippet) bool {
		if language == "" {
			return true
		}
		return snippet.GetVar("language") != "" && snippet.GetVar("language") == language
	}
}

// FilterChain creates a `SnippetFilter` from multiple filters.
// All passed filters are connceted via an AND-Operation
func FilterChain(filters []SnippetFilter) SnippetFilter {
	return func(snippet *snippet.Snippet) bool {
		for _, filter := range filters {
			if filter(snippet) == false {
				return false
			}
		}

		return true
	}
}

// IdFilter creates a filter that filters a snippet based on its `id` property
func IdFilter(id string) SnippetFilter {
	return func(snippet *snippet.Snippet) bool {
		snippetID := snippet.GetVar("id")
		return snippetID != "" && snippetID == id
	}
}

// TagFilter creates a filter that filters a snippet based on its `tag` property.
// As for now, the logic implemented is AND, e.g. a snippet passes the filter if
// it contains every tag in `filtertags`
func TagFilter(filtertags []string) SnippetFilter {
	return func(snippet *snippet.Snippet) bool {
		tags := snippet.GetVar("tags")
		set := make(map[string]struct{})
		for _, tag := range strings.Split(tags, ",") {
			set[tag] = struct{}{}
		}

		for _, filterTag := range filtertags {
			_, present := set[filterTag]
			if present == false {
				return false
			}
		}

		return true
	}
}

// Wildcard creates a filter that lett all snippets slip through
func Wildcard() SnippetFilter {
	return func(snippet *snippet.Snippet) bool {
		return true
	}
}
