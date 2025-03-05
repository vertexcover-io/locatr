package constants

import _ "embed"

// Default cache path
const DEFAULT_CACHE_PATH = ".locatr.cache"

// CHUNK_SIZE is the maximum size of a html chunk
const CHUNK_SIZE = 4000

const CHUNK_OVERLAP = 200

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
