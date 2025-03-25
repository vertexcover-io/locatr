package locatr

import _ "embed"

//go:embed meta/htmlMinifier.js
var HTML_MINIFIER_JS_CONTENT string

const LOCATR_PROMPT = `Your task is to identify the HTML element that matches a user's requirement from a given HTML DOM structure and return its unique_id in a JSON format. The element may not match user's requirement exactly. If the element is not found, provide an appropriate error message in the JSON output.

Each HTML element may contain an attribute called "data-supported-primitives" which indicates its supported interactions. The following attributes determine whether an element is "clickable", "hoverable", "inputable", or "selectable":

1. "clickable": The element supports click interactions and will have "data-supported-primitives" set to "click".
2. "hoverable": The element supports hover interactions and will have "data-supported-primitives" set to "hover".
3. "inputable": The element supports text input interactions and will have "data-supported-primitives" set to "input_text". If this attribute is not present then the input is read-only.
4. "selectable": The element supports selecting options and will have "data-supported-primitives" set to "select_option".

Your output should be in valid JSON format and contain only the required fields.

Output Format:
{
  "locator_id": "str",     // The unique_id of the element that matches the user's requirement.
  "error": "str"           // An appropriate error message if the element is not found.
}

Input Format:
{
  "html_dom": "<!-- Your HTML DOM here -->",
  "user_req": "The user's requirement here"
}
Process the input accordingly and ensure that if the element is not found, the “error” field contains a relevant message.
`

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
