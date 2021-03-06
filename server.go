package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"snippetserver/filters"
	"snippetserver/snipesparsing"
	"snippetserver/snippet"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func processIDs(snippets []*snippet.Snippet) bool {
	processed := false
	for _, snippet := range snippets {
		if snippet.GetVar("id") == "" {
			log.Debug("found snippet without id")
			processed = true
			nano := time.Now().UnixNano()
			sum256 := sha256.Sum256([]byte(strconv.FormatInt(nano, 10)))
			idstring := base64.URLEncoding.EncodeToString(sum256[:])
			snippet.AddVar("id", idstring)
		}
	}

	return processed
}

// createLastSearchFile creates a file. Destination is `snipesserverDir`.
// The created file acts as a buffer and holds the snippetIDs that where found in the
// last invocation of the snippetserver. For multiple found snippets, it will create
// a line for every found snippet in the followign form:
// <nr>: <snippet-ID>
func createLastSearchFile(snipes []*snippet.Snippet, snipesserverDir string) {
	lastSearchFile := snipesserverDir + "/last"
	os.Remove(lastSearchFile)

	file, _ := os.Create(lastSearchFile)
	for i, snippet := range snipes {
		io.WriteString(file, strconv.Itoa(i+1)+": "+snippet.GetVar("id")+"\n")
	}
}

// getSnippetFromLastSearch queries the `lastSearch` file.
// the client provides the number of the snippet-id, she wants to get
func getSnippetFromLastSearch(nr int) *snippet.Snippet {
	file, _ := os.Open(os.Getenv("HOME") + "/.snipeserver/last")

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ":")
		if len(split) != 2 {
			panic("Could not extract id from last seeach")
		}
		i, _ := strconv.Atoi(split[0])
		if i == nr {
			log.Debugf("Snippet nr %s has id %s", split[0], split[1])
			snippets := snippetFinder(filters.IdFilter(strings.Trim(split[1], " ")), "")
			return snippets[0]
		}
	}
	return nil
}

func snippetFinder(filter filters.SnippetFilter, excludeFile string) []*snippet.Snippet {
	snippetFolder := os.Getenv("SNIPES")
	snippets := make([]*snippet.Snippet, 0)
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
			log.Debug("Getting all snippets for id processing")
			snippetsInFile := snipesparsing.ParseSnipe(file, filters.Wildcard())
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
			log.Debug("Now filtering snippets...")
			for _, s := range snippetsInFile {
				if filter(s) {
					snippets = append(snippets, s)
				}
			}
			log.Debugf("Found %d snippets in file %s", len(snippets), path)
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
	nr := flag.Int("nr", -1, "the number from the last search to display")
	verbose := flag.Bool("v", false, "show verbose logs")

	printDesc := flag.Bool("pd", false, "print description")

	flag.Parse()

	if *verbose == true {
		log.SetLevel(log.DebugLevel)
	}

	if len(flag.Args()) > 0 {
		// positional arg
		if flag.Args()[0] == "last" {
			snippet := getSnippetFromLastSearch(*nr)
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

	filterFunctions := make([]filters.SnippetFilter, 0)

	// default filter
	filterFunctions = append(filterFunctions, filters.Wildcard())

	if *id != "" {
		filterFunctions = append(filterFunctions, filters.IdFilter(*id))
	} else {
		if *language != "" {
			filterFunctions = append(filterFunctions, filters.LanguageFilter(*language))
		}
		if *tags != "" {
			filterFunctions = append(filterFunctions, filters.TagFilter(strings.Split(*tags, ",")))
		}
	}

	snippets := snippetFinder(filters.FilterChain(filterFunctions), *exclude)

	if *file != "n/a" && len(snippets) > 1 {
		log.Debug("Can not write snippet to file. More than 1 snippet found")
		return
	}

	if *file != "n/a" {
		fs, _ := os.Create(*file)
		defer fs.Close()
		fs.WriteString(snippets[0].Source)
		return
	}

	multipleSnippets := len(snippets) > 1
	log.Debugf("Total snippets found %d", len(snippets))
	for _, snippet := range snippets {
		rulerNeeded := false
		if multipleSnippets {
			fmt.Printf("[%s]\n", snippet.GetVar("id"))
			rulerNeeded = true
		}
		if *printDesc {
			fmt.Printf("description: %s \n", snippet.GetVar("description"))
			rulerNeeded = true
		}

		if rulerNeeded {
			fmt.Println("------")
		}
		fmt.Println(snippet.Source)
	}

	createLastSearchFile(snippets, snipeserverDir)
}
