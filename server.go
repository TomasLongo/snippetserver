package main

import (
	"bufio"
	"os"
	"strings"
	"fmt"
	"bytes"
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
		trimmed := strings.Trim(scanner.Text(), " ")
		if currentState == START {
			if isFrontMatterString(trimmed) {
				currentState = FM
			}
		} else if currentState == FM {
			if isFrontMatterString(trimmed) {
				fmt.Println("PRocessing source")
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
				sourceBuffer.WriteString(trimmed)
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

func main() {
	file, e := os.Open("/Users/tlongo/go/src/snippetserver/testfiles/test.snipe")
	if e != nil {
		panic(e)
	}

	snippets := GetSnippetsFromFile(file, func(snippet *Snippet) bool {
		return snippet.getVar("language") != "" && snippet.getVar("language") == "java"
	})


	fmt.Println("Found snippetcount: ", len(snippets))

	for _, snippet := range snippets {
		fmt.Println(snippet.Source)
	}
}