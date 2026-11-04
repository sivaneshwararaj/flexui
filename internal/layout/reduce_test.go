package layout

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// TestReduce shrinks a layout case that disagrees with Chromium to a
// minimal tree that still disagrees, using a live headless Chromium
// (scripts/chrome-serve.cjs). It is a debugging tool, skipped unless
// -reduce=CASE is given:
//
//	go test ./internal/layout -run TestReduce -v -args -reduce=random_007
var reduceName = flag.String("reduce", "", "name of a case to minimize against live Chromium")

type oracle struct {
	in  *bufio.Writer
	out *bufio.Scanner
}

func (o *oracle) rects(c layoutCase) map[string]rect {
	b, _ := json.Marshal(caseHTML(c))
	o.in.Write(append(b, '\n'))
	o.in.Flush()
	if !o.out.Scan() {
		panic("chrome-serve died")
	}
	var r map[string]rect
	if err := json.Unmarshal(o.out.Bytes(), &r); err != nil {
		panic(err)
	}
	return r
}

func (o *oracle) differs(t *testing.T, c layoutCase) bool {
	got := runCase(t, c)
	for id, want := range o.rects(c) {
		if !closeRect(got[id], want) {
			return true
		}
	}
	return false
}

func cloneSpec(s *spec) *spec {
	c := *s
	c.kids = nil
	for _, k := range s.kids {
		c.kids = append(c.kids, cloneSpec(k))
	}
	return &c
}

// allSpecs lists every node in pre-order.
func allSpecs(s *spec, out *[]*spec) {
	*out = append(*out, s)
	for _, k := range s.kids {
		allSpecs(k, out)
	}
}

func TestReduce(t *testing.T) {
	if *reduceName == "" {
		t.Skip("use -reduce=CASE")
	}
	var c layoutCase
	for _, x := range allCases() {
		if x.name == *reduceName {
			c = x
		}
	}
	if c.root == nil {
		t.Fatalf("no case %q", *reduceName)
	}
	cmd := exec.Command("node", "../../scripts/chrome-serve.cjs")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	o := &oracle{in: bufio.NewWriter(stdin), out: bufio.NewScanner(stdout)}
	o.out.Buffer(make([]byte, 1<<20), 1<<20)
	if !o.differs(t, c) {
		t.Fatalf("%s matches Chromium", c.name)
	}
	try := func(mut func(root *spec) bool) bool {
		cand := c
		cand.root = cloneSpec(c.root)
		if !mut(cand.root) || !o.differs(t, cand) {
			return false
		}
		c = cand
		return true
	}
	for changed := true; changed; {
		changed = false
		var nodes []*spec
		allSpecs(c.root, &nodes)
		// Remove subtrees.
		for i := range nodes {
			for k := range nodes[i].kids {
				i, k := i, k
				if try(func(root *spec) bool {
					var ns []*spec
					allSpecs(root, &ns)
					p := ns[i]
					p.kids = append(p.kids[:k:k], p.kids[k+1:]...)
					return true
				}) {
					changed = true
					break
				}
			}
			if changed {
				break
			}
		}
		if changed {
			continue
		}
		// Remove declarations, simplify leaves.
		for i := range nodes {
			decls := splitDecls(nodes[i].css)
			for j := range decls {
				i, j := i, j
				if try(func(root *spec) bool {
					var ns []*spec
					allSpecs(root, &ns)
					d := splitDecls(ns[i].css)
					ns[i].css = strings.Join(append(d[:j:j], d[j+1:]...), ";")
					return true
				}) {
					changed = true
					break
				}
			}
			if changed {
				break
			}
			if nodes[i].leaf == leafText && try(func(root *spec) bool {
				var ns []*spec
				allSpecs(root, &ns)
				ns[i].leaf = leafFixed
				return true
			}) {
				changed = true
				break
			}
		}
	}
	got := runCase(t, c)
	want := o.rects(c)
	t.Logf("minimal case (vw=%d vh=%d):\n%s", c.vw, c.vh, dumpSpec(c.root, ""))
	for _, id := range sortedKeys(want) {
		mark := ""
		if !closeRect(got[id], want[id]) {
			mark = "   <-- differs"
		}
		t.Logf("%-6s ours %v chrome %v%s", id, got[id], want[id], mark)
	}
}

func splitDecls(css string) []string {
	var out []string
	for _, d := range strings.Split(css, ";") {
		if strings.TrimSpace(d) != "" {
			out = append(out, d)
		}
	}
	return out
}

func dumpSpec(s *spec, indent string) string {
	kind := "E"
	switch s.leaf {
	case leafFixed:
		kind = fmt.Sprintf("F %dx%d", s.w, s.h)
	case leafText:
		kind = fmt.Sprintf("T %d words %dx%d", s.words, s.w, s.h)
	}
	out := fmt.Sprintf("%s%s [%s] %q\n", indent, s.id, kind, s.css)
	for _, k := range s.kids {
		out += dumpSpec(k, indent+"  ")
	}
	return out
}
