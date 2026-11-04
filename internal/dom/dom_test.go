package dom

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	doc, warns := Parse(`<!doctype html><html><head><style>a{x:y}</style><title>t</title></head>
<body class="app"><div id="a" class="x  y" style="width:1px">hi<span></span></div><blink></blink>
<script>alert(1)</script><style>b{}</style><input type="checkbox"></body></html>`)
	if doc.Root.Tag() != "body" || !doc.Root.HasClass("app") {
		t.Fatalf("root %+v", doc.Root)
	}
	if len(doc.Styles) != 2 || doc.Styles[0] != "a{x:y}" || doc.Styles[1] != "b{}" {
		t.Errorf("styles %q", doc.Styles)
	}
	var tags []string
	for _, c := range doc.Root.Children {
		if c.Kind == ElementNode {
			tags = append(tags, c.Tag())
		}
	}
	if strings.Join(tags, ",") != "div,blink,input" {
		t.Errorf("children %v", tags)
	}
	div := doc.Root.Children[0]
	if div.ID() != "a" || !div.HasClass("y") || div.Attr("style") != "width:1px" || div.ParentElement() == nil {
		t.Errorf("div %+v", div)
	}
	if div.Children[0].Kind != TextNode || div.Children[0].Text != "hi" {
		t.Errorf("text %+v", div.Children[0])
	}
	all := strings.Join(warns, "|")
	for _, w := range []string{"blink", "script", "checkbox"} {
		if !strings.Contains(all, w) {
			t.Errorf("missing warning about %s: %v", w, warns)
		}
	}
	if doc.Root.ParentElement() != nil {
		t.Error("root has a parent")
	}
}

func TestFragment(t *testing.T) {
	doc, warns := Parse(`<div class="a"></div><div class="b"></div>`)
	if len(warns) != 0 || len(doc.Root.Children) != 2 || doc.Root.Tag() != "body" {
		t.Fatalf("%+v %v", doc.Root, warns)
	}
}
