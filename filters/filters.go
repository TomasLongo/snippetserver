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

// TagFilter creates a filter that filters a snippet based on its `tag` property
func TagFilter(filtertags []string) SnippetFilter {
	return func(snippet *snippet.Snippet) bool {
		tags := snippet.GetVar("tags")
		// Implement OR-Logic
		for _, tag := range strings.Split(tags, ",") {
			for _, filterTag := range filtertags {
				if filterTag == tag {
					return true
				}
			}
		}

		return false
	}
}
