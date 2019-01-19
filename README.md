# SnippetServer

Centralized Snippetrepo to manage usefull textsnippets.

## Usage

### The snipes files

Snippets (or snipes) are stored in normal textfiles. Every file can contain multiple snippets which every snippet having the following form

```text
---
[metadata]
---
[content]
```

Borrowing the concept from yaml?? a snippet is made up of its metadata (e.g. the frontmatter) and its actual content. The metadata is simply a collection of key-value-pairs that further describe the content. A simple snippet containing some go-code could be

```yaml
---
language: go
description: Another Test Snippet
tags: go
id: 1234
---
func isFrontMatterString(s string) bool {
    return strings.HasPrefix(s, "---")
}
```

> Adding new snippets is simply a matter of editing the snipes-files

### The snipes file directory

The snippetserver will parse every file file ending with `.snipes` inside the directory that is specifiied by tbe einvoronment variable `SNIPES`

### Query

The snippetserver has a rich set of filters to browse and find stored snippets.

#### Filter snippets

Using the cli you can filter snippets by the following properties:

* by tags
* by language
* by id

```bash
snippetserver -tags=template,bootstrap
snippetserver -lang=go
```

#### Show Description

By default, the server will output a snippet's content along with its id. To also print its description property use the flag `-dp`