package constants

import _ "embed"

// Default cache path
const DEFAULT_CACHE_PATH = ".locatr.cache"

// DEFAULT_CHUNK_SIZE is the default maximum size of a dom chunk
const DEFAULT_CHUNK_SIZE = 4000

// DEFAULT_CHUNK_OVERLAP is the default overlap between chunks
const DEFAULT_CHUNK_OVERLAP = 200

// DEFAULT_MAX_ATTEMPTS is the default maximum number of attempts that can be made to process a single Id completion request
const DEFAULT_MAX_ATTEMPTS = 3

// DEFAULT_CHUNKS_PER_ATTEMPT is the default number of chunks than can be processed in a single attempt of Id completion request
const DEFAULT_CHUNKS_PER_ATTEMPT = 3

// DEFAULT_TOP_N is the default number of chunks that will be returned from the reranker
const DEFAULT_TOP_N = 10

//go:embed meta/script.js
var JS_CONTENT string

// HTML_SEPARATORS used by html chunk splitter
var HTML_SEPARATORS = []string{
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
	"",
}

const DEFAULT_LOAD_EVENT_TIMEOUT float64 = 30000.0
