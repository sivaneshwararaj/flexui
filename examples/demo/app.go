package main

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/sivaneshwararaj/flexui"
)

// Contact is one entry of the demo's address book.
type Contact struct {
	ID    int
	Name  string
	Email string
	Notes string
}

// Demo is the application state. The view is a pure function of it.
type Demo struct {
	App      *flexui.App
	contacts []Contact
	nextID   int
	query    string
	selected int // contact ID, 0 = none
	// Draft is the form being edited (a copy of the selected contact).
	draft   Contact
	dirty   bool
	confirm bool // the delete dialog is open
	status  string
}

var seed = []string{
	"Ada Lovelace|ada@analytical.engine|First published algorithm.",
	"Alan Turing|alan@bletchley.uk|Ask about the halting problem.",
	"Barbara Liskov|barbara@mit.edu|Substitution principle.",
	"Claude Shannon|claude@bell-labs.com|Information theory; juggling.",
	"Donald Knuth|don@stanford.edu|Send checks for errata.",
	"Edsger Dijkstra|edsger@utexas.edu|Prefers handwritten notes.",
	"Frances Allen|fran@ibm.com|Compiler optimization pioneer.",
	"Grace Hopper|grace@navy.mil|Nanosecond wire on the desk.",
	"Hedy Lamarr|hedy@hollywood.com|Frequency hopping.",
	"John McCarthy|john@stanford.edu|Lisp; garbage collection.",
	"Katherine Johnson|katherine@nasa.gov|Trajectory calculations.",
	"Ken Thompson|ken@bell-labs.com|Unix, UTF-8, Go.",
	"Margaret Hamilton|margaret@mit.edu|Apollo flight software.",
	"Radia Perlman|radia@dec.com|Spanning tree protocol.",
}

// NewDemo builds the app: state, view and handlers, at 760x480.
func NewDemo() *Demo { return newDemo(flexui.Options{Width: 760, Height: 480}, 0) }

// newDemo builds the app with the given size and scale, and extra
// generated contacts (for benchmarks).
func newDemo(o flexui.Options, extra int) *Demo {
	d := &Demo{nextID: 1}
	for _, s := range seed {
		f := strings.Split(s, "|")
		d.contacts = append(d.contacts, Contact{ID: d.nextID, Name: f[0], Email: f[1], Notes: f[2]})
		d.nextID++
	}
	for i := 0; i < extra; i++ {
		d.contacts = append(d.contacts, Contact{ID: d.nextID, Name: fmt.Sprintf("Zed Person %03d", i),
			Email: fmt.Sprintf("person%03d@example.com", i), Notes: "Generated."})
		d.nextID++
	}
	d.selectContact(1)
	o.Title = "flexui demo: contacts"
	d.App = flexui.New(o)
	tmpl := template.Must(template.New("demo").Parse(view))
	d.App.SetViewFunc(func() string {
		var b bytes.Buffer
		if err := tmpl.Execute(&b, d); err != nil {
			return "<div>template error: " + template.HTMLEscapeString(err.Error()) + "</div>"
		}
		return b.String()
	}, css)
	h := d.App.Handle
	h("Search", func(c *flexui.Ctx) { d.query = c.Value })
	h("Select", func(c *flexui.Ctx) {
		var id int
		fmt.Sscan(c.Attr("data-id"), &id)
		d.selectContact(id)
	})
	h("EditName", func(c *flexui.Ctx) { d.draft.Name = c.Value; d.dirty = true })
	h("EditEmail", func(c *flexui.Ctx) { d.draft.Email = c.Value; d.dirty = true })
	h("EditNotes", func(c *flexui.Ctx) { d.draft.Notes = c.Value; d.dirty = true })
	h("Save", func(c *flexui.Ctx) { d.save() })
	h("Revert", func(c *flexui.Ctx) { d.selectContact(d.selected) })
	h("Add", func(c *flexui.Ctx) {
		d.add()
		c.Focus("name") // after the re-render: type the new name right away
	})
	h("AskDelete", func(c *flexui.Ctx) { d.confirm = true })
	h("CancelDelete", func(c *flexui.Ctx) { d.confirm = false })
	h("ConfirmDelete", func(c *flexui.Ctx) { d.delete() })
	h("Backdrop", func(c *flexui.Ctx) {
		if c.TargetID == "backdrop" { // a click outside the dialog
			d.confirm = false
		}
	})
	h("Key", func(c *flexui.Ctx) {
		if c.Key == "Escape" && d.confirm {
			d.confirm = false
		}
	})
	return d
}

func (d *Demo) find(id int) int {
	for i, c := range d.contacts {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// selectContact shows contact id in the form. It clears the status line:
// a message like "Saved X." is about the previous selection. (Add and
// delete set their own status after selecting.)
func (d *Demo) selectContact(id int) {
	d.selected, d.dirty = id, false
	d.status = ""
	if i := d.find(id); i >= 0 {
		d.draft = d.contacts[i]
	} else {
		d.draft = Contact{}
	}
}

func (d *Demo) save() {
	i := d.find(d.selected)
	if i < 0 || !d.dirty {
		return
	}
	if strings.TrimSpace(d.draft.Name) == "" {
		d.status = "A name is required."
		return
	}
	d.contacts[i] = d.draft
	sort.SliceStable(d.contacts, func(a, b int) bool { return d.contacts[a].Name < d.contacts[b].Name })
	d.dirty = false
	d.status = "Saved " + d.draft.Name + "."
}

func (d *Demo) add() {
	c := Contact{ID: d.nextID, Name: "New contact"}
	d.nextID++
	d.contacts = append(d.contacts, c)
	sort.SliceStable(d.contacts, func(a, b int) bool { return d.contacts[a].Name < d.contacts[b].Name })
	d.query = ""
	d.selectContact(c.ID)
	d.status = "Added a contact."
}

func (d *Demo) delete() {
	i := d.find(d.selected)
	d.confirm = false
	if i < 0 {
		return
	}
	// Select the next visible contact (or the previous one at the end).
	vis := d.Visible()
	next := 0
	for k, c := range vis {
		if c.ID == d.selected {
			if k+1 < len(vis) {
				next = vis[k+1].ID
			} else if k > 0 {
				next = vis[k-1].ID
			}
		}
	}
	name := d.contacts[i].Name
	d.contacts = append(d.contacts[:i], d.contacts[i+1:]...)
	d.selectContact(next)
	d.status = "Deleted " + name + "."
}

// Template helpers (methods used by the view).

// Visible returns the contacts matching the search query.
func (d *Demo) Visible() []Contact {
	var out []Contact
	q := strings.ToLower(strings.TrimSpace(d.query))
	for _, c := range d.contacts {
		if q == "" || strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.Email), q) {
			out = append(out, c)
		}
	}
	return out
}

func (d *Demo) Query() string      { return d.query }
func (d *Demo) Selected() int      { return d.selected }
func (d *Demo) Draft() Contact     { return d.draft }
func (d *Demo) Dirty() bool        { return d.dirty }
func (d *Demo) Confirm() bool      { return d.confirm }
func (d *Demo) Status() string     { return d.status }
func (d *Demo) Count() int         { return len(d.contacts) }
func (d *Demo) HasSelection() bool { return d.find(d.selected) >= 0 }

// Initials returns up to two initials for the avatar.
func (c Contact) Initials() string {
	var out []rune
	for _, f := range strings.Fields(c.Name) {
		out = append(out, []rune(f)[0])
		if len(out) == 2 {
			break
		}
	}
	return strings.ToUpper(string(out))
}
