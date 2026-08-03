// Package markdown renders untrusted Markdown to sanitized HTML.
//
// Input is always treated as untrusted: the parser escapes raw HTML and the
// rendered output is passed through an allowlist sanitizer, so AI- and
// user-authored Markdown can be embedded in a page without introducing script
// or dangerous-URI injection.
package markdown

import (
	"bytes"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// renderer parses CommonMark plus the GFM extension (tables, strikethrough,
// task lists, autolinks). Raw HTML is left escaped because WithUnsafe is not
// enabled. It is safe for concurrent use.
var renderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

// policy is an allowlist sanitizer for user-generated content. It permits
// common formatting and links while stripping scripts, event handlers, and
// unsafe URL schemes. It is safe for concurrent use once constructed.
var policy = bluemonday.UGCPolicy()

// ToHTML renders Markdown to sanitized HTML safe for embedding in a page.
// If parsing fails, it falls back to the escaped plain text of the input.
func ToHTML(md string) template.HTML {
	var buf bytes.Buffer
	if err := renderer.Convert([]byte(md), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(md))
	}
	return template.HTML(policy.SanitizeBytes(buf.Bytes()))
}
