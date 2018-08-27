package main

import (
	"bufio"
	"os"
	"strings"
	"fmt"
	"bytes"
	"flag"
	"path/filepath"
)

type Snippet struct {
	Source string
	variables map[string]string
}

func (s *Snippet)addVar(key string, value string) {
	s.variables[key] = value
}

func (s *Snippet)getVar(key string) string {
	return s.variables[key]
}

type State int
const (
	FM State = 1
	SOURCE State = 2
	START State = 3
)

var currentState = START
var sourceBuffer = new(bytes.Buffer)

func NewSnippet() *Snippet {
	snippet := new(Snippet)
	snippet.variables = make(map[string]string)

	return snippet
}

type snippetFilter func(snippet *Snippet) bool

func GetSnippetsFromFile(file *os.File, filter snippetFilter) []*Snippet {
	snippets := make([]*Snippet, 0)
	snippet := NewSnippet()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		untrimmed := scanner.Text()
		trimmed := strings.Trim(untrimmed, " ")
		if currentState == START {
			if isFrontMatterString(trimmed) {
				currentState = FM
			}
		} else if currentState == FM {
			if isFrontMatterString(trimmed) {
				currentState = SOURCE
				sourceBuffer.Reset()
			} else {
				tokens := strings.Split(trimmed, ":")

				key := strings.Trim(tokens[0], " ")
				value := strings.Trim(tokens[1], " ")

				snippet.addVar(key, value)
			}
		} else if currentState == SOURCE {
			if isFrontMatterString(trimmed) {
				// new snippet
				currentState = FM
				snippet.Source = sourceBuffer.String()
				if filter(snippet) {
					snippets = append(snippets, snippet)
				}

				snippet = NewSnippet()
			} else {
				sourceBuffer.WriteString(untrimmed)
				sourceBuffer.WriteString("\n")
			}
		}
	}

	snippet.Source = sourceBuffer.String()

	if filter(snippet){
		snippets = append(snippets, snippet)
	}

	return snippets
}

func isFrontMatterString(s string) bool {
	return strings.HasPrefix(s, "---")
}

func languageFilter(language string) snippetFilter {
	return func(snippet *Snippet) bool {
		if language == "" {
			return true
		}
		return snippet.getVar("language") != "" && snippet.getVar("language") == language
	}
}

func main() {
	language := flag.String("lang", "", "the language to filter for")
	exclude := flag.String("x", "", "the file, that should be excluded")

	flag.Parse()

	snippets := make([]*Snippet, 0)
	filepath.Walk("/Users/tlongo/go/src/snippetserver/testfiles", func(path string, f os.FileInfo, err error) error {
		if f.Name() == *exclude {
			return nil
		}
		if !f.IsDir() && strings.HasSuffix(path, ".snipe") {
			fmt.Println("Parsing ", path, " for snippets")
			file, e := os.Open(path)
			defer file.Close()
			if e != nil {
				panic(e)
			}
			snippets = append(snippets, GetSnippetsFromFile(file, languageFilter(*language))...)
		}
		return nil
	})

	fmt.Println("Found: ", len(snippets), " snippets")
	for _, snippet := range snippets {
		fmt.Println(snippet.Source)
	}
}