package snipesparsing

import (
	"bufio"
	"bytes"
	"os"
	"snippetserver/filters"
	"snippetserver/snippet"
	"strings"

	log "github.com/sirupsen/logrus"
)

type State int

const (
	FM     State = 1
	SOURCE State = 2
	START  State = 3
)

var currentState = START
var sourceBuffer = new(bytes.Buffer)

func ParseSnipe(file *os.File, filter filters.SnippetFilter) []*snippet.Snippet {
	snippets := make([]*snippet.Snippet, 0)
	snip := snippet.NewSnippet()

	scanner := bufio.NewScanner(file)
	currentState = START

	for scanner.Scan() {
		untrimmed := scanner.Text()
		trimmed := strings.Trim(untrimmed, " ")
		if currentState == START {
			if isFrontMatterString(trimmed) {
				log.Debug("Found FM. Start parsing attributes")
				currentState = FM
			}
		} else if currentState == FM {
			if isFrontMatterString(trimmed) {
				log.Debug("End of FM. Start parsing source")
				currentState = SOURCE
				sourceBuffer.Reset()
			} else {
				tokens := strings.Split(trimmed, ":")

				key := strings.Trim(tokens[0], " ")
				value := strings.Trim(tokens[1], " ")
				log.Debugf("%s: %s", key, value)

				snip.AddVar(key, value)
			}
		} else if currentState == SOURCE {
			if isFrontMatterString(trimmed) {
				log.Debug("Found new snippet")
				// new snip
				currentState = FM
				snip.Source = sourceBuffer.String()
				if filter(snip) {
					log.Debugf("Adding snippet: %s", snip.GetVar("id"))
					snippets = append(snippets, snip)
					log.Debugf("Snippetcount in file: %d", len(snippets))
				}

				snip = snippet.NewSnippet()
			} else {
				sourceBuffer.WriteString(untrimmed)
				sourceBuffer.WriteString("\n")
			}
		}
	}

	snip.Source = sourceBuffer.String()

	if filter(snip) {
		log.Debugf("Adding snippet: %s", snip.GetVar("id"))
		snippets = append(snippets, snip)
		log.Debugf("Snippetcount in file: %d", len(snippets))
	}

	return snippets
}

func isFrontMatterString(s string) bool {
	return strings.HasPrefix(s, "---")
}
