package locatr

func GetUniqueStringArray(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

var seperators = []string{
	"<body",
	"<div",
	"<p",
	"<br",
	"<li",
	"<h1",
	"<h2",
	"<h3",
	"<h4",
	"<h5",
	"<h6",
	"<span",
	"<table",
	"<tr",
	"<td",
	"<th",
	"<ul",
	"<ol",
	"<header",
	"<footer",
	"<nav",
	"<head",
	"<style",
	"<script",
	"<meta",
	"<title",
}

func split(html string) []string {
	currentChunk := ""
	currentIndex := 0
	chunks := []string{}
	for {
		if currentIndex == len(html) {
			break
		}
		for _, sep := range seperators {
			sepLength := len(sep)
			if currentIndex+sepLength > len(html) {
				continue
			}
			if html[currentIndex:currentIndex+sepLength] == sep {
				if len(currentChunk)+len(sep) > 4000 {
					continue
				}
			}
		}
		if len(currentChunk) > 4000 {
			chunks = append(chunks, currentChunk)
			currentChunk = ""
			continue
		}
		currentChunk += string(html[currentIndex])
		currentIndex++
	}
	return chunks
}
