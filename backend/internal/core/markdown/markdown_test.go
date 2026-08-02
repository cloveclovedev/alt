package markdown

import (
	"strings"
	"testing"
)

func TestToHTMLRendersCommonMark(t *testing.T) {
	got := string(ToHTML("# Title\n\n- one\n- two\n"))
	if !strings.Contains(got, "<h1>Title</h1>") {
		t.Errorf("heading not rendered: %q", got)
	}
	if !strings.Contains(got, "<li>one</li>") {
		t.Errorf("list item not rendered: %q", got)
	}
}

func TestToHTMLEscapesRawHTML(t *testing.T) {
	got := string(ToHTML("<script>alert(1)</script>\n"))
	if strings.Contains(got, "<script>") {
		t.Errorf("raw script tag survived: %q", got)
	}
}

func TestToHTMLStripsDangerousLinkScheme(t *testing.T) {
	got := string(ToHTML("[x](javascript:alert(1))\n"))
	if strings.Contains(got, "javascript:") {
		t.Errorf("javascript URI survived: %q", got)
	}
}

func TestToHTMLKeepsSafeLink(t *testing.T) {
	got := string(ToHTML("[issue](https://example.com/1)\n"))
	if !strings.Contains(got, `href="https://example.com/1"`) {
		t.Errorf("safe link not preserved: %q", got)
	}
}
