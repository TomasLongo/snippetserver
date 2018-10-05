package main

import (
	"bufio"
	"os"
	"strings"
	"fmt"
	"bytes"
	"flag"
	"path/filepath"
	"crypto/sha256"
	"time"
	"encoding/base64"
	"strconv"
	"io"
)

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

func filterChain(filters []snippetFilter) snippetFilter {
	return func(snippet *Snippet) bool {
		for _, filter := range filters {
			if filter(snippet) == false {
				return false
			}
		}

		return true
	}
}

func tagFilter(filtertags []string) snippetFilter {
	return func(snippet *Snippet) bool {
		tags := snippet.getVar("tags")
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

func processIDs(snippets []*Snippet) bool {
	fmt.Println("processing ids")
	processed := false
	for _, snippet := range snippets {
		if snippet.getVar("id") == "" {
			fmt.Println("found snippet without id")
			processed = true
			nano := time.Now().UnixNano()
			sum256 := sha256.Sum256([]byte(strconv.FormatInt(nano, 10)))
			idstring := base64.URLEncoding.EncodeToString(sum256[:])
			snippet.addVar("id", idstring)
		}
	}

	return processed
}

func main() {
	language := flag.String("lang", "", "the language to filter for")
	exclude := flag.String("x", "", "the file, that should be excluded")
	tags := flag.String("tag", "", "the tag to filter for")
	file := flag.String("file", "n/a", "the file to write the snippet to")

	flag.Parse()

	snippetFolder := os.Getenv("SNIPES")
	snippets := make([]*Snippet, 0)

	filters := make([]snippetFilter, 0)

	filters = append(filters, func(snippet *Snippet) bool { return true })
	if *language != "" {
		filters = append(filters, languageFilter(*language))
	}
	if *tags != "" {
		filters = append(filters, tagFilter(strings.Split(*tags, ",")))
	}
	filepath.Walk(snippetFolder, func(path string, f os.FileInfo, err error) error {
		if f.Name() == *exclude {
			return nil
		}
		if !f.IsDir() && strings.HasSuffix(path, ".snipe") {
			file, e := os.Open(path)
			defer file.Close()
			if e != nil {
				panic(e)
			}
			snippetsInFile := GetSnippetsFromFile(file, filterChain(filters))
			if processIDs(snippetsInFile) == true {
				// backup current snipes file
				backupFile, _ := os.Create(path + ".bk")
				file.Seek(0, io.SeekStart)
				_, bkCreateError := io.Copy(backupFile, file)
				if  bkCreateError != nil {
					panic(bkCreateError)
				}
				backupFile.Sync()
				backupFile.Close()

				// rm old snipes file
				file.Close()
				os.Remove(path)

				// Write new snipes file with ids
				newFile, err := os.Create(path)
				if err != nil {
					panic(err)
				}
				for _, snippet := range snippetsInFile {
					newFile.WriteString(snippet.String())
				}
				newFile.Sync()
				newFile.Close()
			}
			snippets = append(snippets, snippetsInFile...)
		}
		return nil
	})

	if *file != "n/a" && len(snippets) > 1 {
		fmt.Println("Can not write snippet to file. More than 1 snippet found")
		return
	}

	if *file != "n/a" {
		fs,_ := os.Create(*file)
		defer fs.Close()
		fs.WriteString(snippets[0].Source)
		return
	}

	for _, snippet := range snippets {
		fmt.Println(snippet.Source)
	}
}