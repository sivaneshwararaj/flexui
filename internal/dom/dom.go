// Package dom builds flexui's node tree from HTML.
//
// The document is parsed with golang.org/x/net/html (a full HTML5
// parser). The `<body>` element is the root of the tree (spec/SUBSET.md
// §1); `<html>` and `<head>` are not part of it. `<style>` elements
// anywhere in the document are collected, in document order, as style
// sheets.
package dom

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"

	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Kind distinguishes element and text nodes.
type Kind uint8

const (
	ElementNode Kind = iota
	TextNode
)

// Attr is one attribute.
type Attr struct{ Name, Value string }

// Node is an element or a text node.
type Node struct {
	Kind     Kind
	TagName  string // lower-case, elements only
	Attrs    []Attr // in source order
	Text     string // text nodes only (raw, whitespace preserved)
	Parent   *Node
	Children []*Node

	id      string
	classes []string
	// Dynamic state for :hover/:focus/:active, set by the app (M6).
	Dyn style.State
}

// Document is a parsed document.
type Document struct {
	Root   *Node    // the <body> element
	Styles []string // contents of <style> elements, in document order
}

// Elements supported by the subset (spec/SUBSET.md §1). Others behave as
// div and produce a warning.
var known = map[string]bool{
	"body": true, "div": true, "span": true, "p": true, "h1": true, "h2": true, "h3": true,
	"button": true, "input": true, "img": true, "ul": true, "li": true, "a": true,
	"strong": true, "b": true, "em": true, "i": true, "br": true,
}

// Handler attributes (spec/SUBSET.md §8). Other on* attributes warn.
var handlerAttrs = map[string]bool{
	"onclick": true, "onkeydown": true, "onfocus": true, "onblur": true,
	"oninput": true, "onchange": true,
}

// Elements that are never rendered.
var skipped = map[string]bool{
	"head": true, "title": true, "meta": true, "link": true, "style": true,
	"script": true, "template": true, "noscript": true,
}

// Parse parses an HTML document or fragment.
func Parse(src string) (*Document, []string) {
	var warns []string
	top, err := html.Parse(strings.NewReader(src))
	if err != nil { // x/net/html only fails on reader errors
		return &Document{Root: &Node{Kind: ElementNode, TagName: "body"}}, []string{err.Error()}
	}
	doc := &Document{}
	var body *html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "style":
				var sb strings.Builder
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						sb.WriteString(c.Data)
					}
				}
				doc.Styles = append(doc.Styles, sb.String())
			case "script":
				warns = append(warns, "<script> is not supported and was ignored")
			case "body":
				if body == nil {
					body = n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(top)
	if body == nil { // cannot happen with the HTML5 algorithm
		body = &html.Node{Type: html.ElementNode, Data: "body"}
	}
	doc.Root = convert(body, nil, &warns)
	return doc, warns
}

func convert(h *html.Node, parent *Node, warns *[]string) *Node {
	n := &Node{Kind: ElementNode, TagName: strings.ToLower(h.Data), Parent: parent}
	if !known[n.TagName] {
		*warns = append(*warns, fmt.Sprintf("<%s> is not supported; it behaves as <div>", n.TagName))
	}
	for _, a := range h.Attr {
		n.Attrs = append(n.Attrs, Attr{a.Key, a.Val})
		if strings.HasPrefix(a.Key, "on") && !handlerAttrs[a.Key] {
			*warns = append(*warns, fmt.Sprintf("handler attribute %s is not supported (only onclick, onkeydown, onfocus, onblur, oninput, onchange)", a.Key))
		}
		switch a.Key {
		case "id":
			n.id = a.Val
		case "class":
			n.classes = strings.Fields(a.Val)
		}
	}
	if (n.TagName == "button" || n.TagName == "input") && hasAttr(h, "disabled") {
		n.Dyn |= style.StateDisabled
	}
	if n.TagName == "input" {
		if t := n.Attr("type"); t != "" && strings.ToLower(t) != "text" {
			*warns = append(*warns, fmt.Sprintf("<input type=%q> is not supported; treated as text", t))
		}
	}
	for c := h.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.ElementNode:
			if skipped[c.Data] {
				continue
			}
			n.Children = append(n.Children, convert(c, n, warns))
		case html.TextNode:
			n.Children = append(n.Children, &Node{Kind: TextNode, Text: c.Data, Parent: n})
		}
	}
	return n
}

// Attr returns the value of the named attribute, or "".
func (n *Node) Attr(name string) string {
	for _, a := range n.Attrs {
		if a.Name == name {
			return a.Value
		}
	}
	return ""
}

// Tag implements style.Element.
func (n *Node) Tag() string { return n.TagName }

// ID implements style.Element.
func (n *Node) ID() string { return n.id }

// HasClass implements style.Element.
func (n *Node) HasClass(c string) bool {
	for _, x := range n.classes {
		if x == c {
			return true
		}
	}
	return false
}

// ParentElement implements style.Element.
func (n *Node) ParentElement() style.Element {
	if n.Parent == nil {
		return nil
	}
	return n.Parent
}

// State implements style.Element.
func (n *Node) State() style.State { return n.Dyn }

// IsWhitespace reports whether a text node contains only whitespace.
func (n *Node) IsWhitespace() bool { return strings.TrimSpace(n.Text) == "" }

func hasAttr(h *html.Node, name string) bool {
	for _, a := range h.Attr {
		if a.Key == name {
			return true
		}
	}
	return false
}

// Disabled reports whether n is a disabled button or input.
func (n *Node) Disabled() bool { return n.Dyn&style.StateDisabled != 0 }

// FindID returns the first element with the given id at or below n.
func (n *Node) FindID(id string) *Node {
	if n.Kind == ElementNode && n.id == id {
		return n
	}
	for _, c := range n.Children {
		if r := c.FindID(id); r != nil {
			return r
		}
	}
	return nil
}
