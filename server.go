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

func idFilter(id string) snippetFilter {
	return func(snippet *Snippet) bool {
		snippetID := snippet.getVar("id")
		return snippetID != "" && snippetID == id
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

// createLastSearchFile creates a file. Destination is `snipesserverDir`.
// The created file acts as a buffer and holds the snippetIDs that where found in the
// last invocation of the snippetserver. For multiple found snippets, it will create
// a line for every found snippet in the followign form:
// <nr>: <snippet-ID>
func createLastSearchFile(snipes []*Snippet, snipesserverDir string) {
	lastSearchFile := snipesserverDir + "/.last"
	os.Remove(lastSearchFile)

	file, _ := os.Create(lastSearchFile)
	for i, snippet := range snipes {
		io.WriteString(file, strconv.Itoa(i+1) + ": " +  snippet.getVar("id") + "\n")
	}
}

// getSnippetFromLastSearch queries the `lastSearch` file.
// the client provides the number of the snippet-id, she wants to get
func getSnippetFromLastSearch(nr int) *Snippet {
	file, _ := os.Open(os.Getenv("HOME") + "/.snipeserver/.last")

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ":")
		i, _ := strconv.Atoi(split[0])
		if i == nr {
			snippets := snippetFinder(idFilter(strings.Trim(split[1], " ")), "")
			return snippets[0]
		}
	}

	return nil
}

func snippetFinder(filter snippetFilter, excludeFile string) []*Snippet {
	snippetFolder := os.Getenv("SNIPES")
	snippets := make([]*Snippet, 0)
	e := filepath.Walk(snippetFolder, func(path string, f os.FileInfo, err error) error {
		if f.Name() == excludeFile {
			return nil
		}
		if !f.IsDir() && strings.HasSuffix(path, ".snipe") {
			file, e := os.Open(path)
			defer file.Close()
			if e != nil {
				panic(e)
			}
			snippetsInFile := GetSnippetsFromFile(file, filter)
			if processIDs(snippetsInFile) == true {
				// backup current snipes file
				backupFile, _ := os.Create(path + ".bk")
				file.Seek(0, io.SeekStart)
				_, bkCreateError := io.Copy(backupFile, file)
				if bkCreateError != nil {
					panic(bkCreateError)
				}
				backupFile.Sync()
				backupFile.Close()

				// rm old snipes file
				file.Close()
				os.Remove(path)

				// Write new snipes file with ids
				// TODO: This is FATAL for it will populate the snipes file with found snipes *only*
				// We have to somehow inject the id into the present snipes file. Maybe while processing the IDs??
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

	if e != nil {
		panic(fmt.Sprintf("Could not walk %s. Error: \n%s", snippetFolder, e.Error()))
	}

	return snippets
}

func main() {
	language := flag.String("lang", "", "the language to filter for")
	exclude := flag.String("x", "", "the file, that should be excluded")
	tags := flag.String("tag", "", "the tag to filter for")
	file := flag.String("file", "n/a", "the file to write the snippet to")
	id := flag.String("id", "", "the id of the snippet")
	nr := flag.String("nr", "", "the number from the last search to display")

	flag.Parse()

	if len(flag.Args()) > 0 {
		// positional arg
		if flag.Args()[0] == "last" {
			i, _ := strconv.Atoi(*nr)
			snippet := getSnippetFromLastSearch(i)
			fmt.Println(snippet.Source)
			return
		}
	}

	// create snipesserver dir in users home if needed
	snipeserverDir := os.Getenv("HOME") + "/.snipeserver"
	_, e := os.Stat(snipeserverDir)
	if os.IsNotExist(e) {
		os.Mkdir(snipeserverDir, os.ModePerm)
	}

	filters := make([]snippetFilter, 0)

	// default filter
	filters = append(filters, func(snippet *Snippet) bool { return true })

	if *id != "" {
		filters = append(filters, idFilter(*id))
	} else {
		if *language != "" {
			filters = append(filters, languageFilter(*language))
		}
		if *tags != "" {
			filters = append(filters, tagFilter(strings.Split(*tags, ",")))
		}
	}

	snippets := snippetFinder(filterChain(filters), *exclude)

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

	multipleSnippets := len(snippets) > 0
	for _, snippet := range snippets {
		if multipleSnippets {
			fmt.Printf("[%s]\n", snippet.getVar("id"))
		}
		fmt.Println(snippet.Source)
	}

	createLastSearchFile(snippets, snipeserverDir)
}