package main

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/sivaneshwararaj/flexui"
)

// Product is one row of the order tab. Prices are in cents.
type Product struct {
	ID       int
	Name     string
	Category string
	Price    int
	Qty      int
	Selected bool
}

// Showcase is the app's state; the view template reads it through the
// methods below and handlers change it.
type Showcase struct {
	App *flexui.App

	tab string

	// Order tab.
	products   []Product
	categories map[string]bool // category filter chips (none on = all)
	taxRate    int             // percent
	coupon     string
	checkout   bool
	placed     string

	// Calculator tab.
	display string // what the display shows
	acc     float64
	op      string
	fresh   bool // the next digit starts a new number

	// Languages tab.
	typed string
}

// NewShowcase builds the app.
func NewShowcase() *Showcase {
	s := &Showcase{tab: "order", taxRate: 5, display: "0", fresh: true, categories: map[string]bool{}}
	for i, p := range []struct {
		name, cat string
		price     int
	}{
		{"Apples (1 kg)", "Fruit", 349}, {"Bananas (6)", "Fruit", 199}, {"Mango", "Fruit", 129},
		{"Sourdough loaf", "Bakery", 450}, {"Croissant", "Bakery", 175}, {"Bagels (4)", "Bakery", 320},
		{"Milk (1 l)", "Dairy", 119}, {"Greek yogurt", "Dairy", 259}, {"Cheddar", "Dairy", 489},
		{"Coffee beans", "Pantry", 1099}, {"Olive oil", "Pantry", 899}, {"Basmati rice (2 kg)", "Pantry", 649},
	} {
		s.products = append(s.products, Product{ID: i + 1, Name: p.name, Category: p.cat, Price: p.price, Qty: 1})
	}
	s.App = flexui.New(flexui.Options{Title: "flexui showcase", Width: 860, Height: 560})
	tmpl := template.Must(template.New("view").Funcs(template.FuncMap{
		"list": func(xs ...any) []any { return xs },
	}).Parse(view))
	s.App.SetViewFunc(func() string {
		var b bytes.Buffer
		if err := tmpl.Execute(&b, s); err != nil {
			return "<div>" + template.HTMLEscapeString(err.Error()) + "</div>"
		}
		return b.String()
	}, css)
	h := s.App.Handle
	h("Tab", func(c *flexui.Ctx) { s.tab = c.Attr("data-tab") })

	// Order.
	h("Toggle", func(c *flexui.Ctx) {
		if p := s.product(c.Attr("data-id")); p != nil {
			p.Selected = !p.Selected
		}
	})
	h("Inc", func(c *flexui.Ctx) {
		if p := s.product(c.Attr("data-id")); p != nil && p.Qty < 99 {
			p.Qty++
			p.Selected = true
		}
		c.StopPropagation() // not a row click
	})
	h("Dec", func(c *flexui.Ctx) {
		if p := s.product(c.Attr("data-id")); p != nil && p.Qty > 1 {
			p.Qty--
		}
		c.StopPropagation()
	})
	h("Filter", func(c *flexui.Ctx) {
		cat := c.Attr("data-cat")
		s.categories[cat] = !s.categories[cat]
	})
	h("SelectAll", func(*flexui.Ctx) {
		for i := range s.products {
			if s.shown(s.products[i]) {
				s.products[i].Selected = true
			}
		}
	})
	h("Clear", func(*flexui.Ctx) {
		for i := range s.products {
			s.products[i].Selected = false
		}
		s.placed = ""
	})
	h("Tax", func(c *flexui.Ctx) { s.taxRate, _ = strconv.Atoi(c.Attr("data-rate")) })
	h("Coupon", func(c *flexui.Ctx) { s.coupon = c.Value })
	h("Checkout", func(*flexui.Ctx) {
		if s.Count() > 0 {
			s.checkout = true
		}
	})
	h("Cancel", func(*flexui.Ctx) { s.checkout = false })
	h("Backdrop", func(c *flexui.Ctx) {
		if c.TargetID == "backdrop" { // a click outside the dialog
			s.checkout = false
		}
	})
	h("Place", func(*flexui.Ctx) {
		s.placed = fmt.Sprintf("Order placed: %d items, %s.", s.Count(), money(s.Total()))
		s.checkout = false
		for i := range s.products {
			s.products[i].Selected, s.products[i].Qty = false, 1
		}
	})
	h("Key", func(c *flexui.Ctx) {
		if c.Key == "Escape" && s.checkout {
			s.checkout = false
		}
	})

	// Calculator.
	h("CalcButton", func(c *flexui.Ctx) { s.press(c.Attr("data-k")) })
	h("CalcKey", func(c *flexui.Ctx) {
		k := map[string]string{"Enter": "=", "Backspace": "⌫", "Escape": "C", "*": "×", "/": "÷", "-": "−"}[c.Key]
		if k == "" {
			k = c.Key
		}
		if strings.Contains("0123456789.+−×÷=C⌫%", k) && k != "" {
			s.press(k)
			c.PreventDefault()
		}
	})

	// Languages.
	h("Typed", func(c *flexui.Ctx) { s.typed = c.Value })
	return s
}

func (s *Showcase) product(id string) *Product {
	n, _ := strconv.Atoi(id)
	for i := range s.products {
		if s.products[i].ID == n {
			return &s.products[i]
		}
	}
	return nil
}

func (s *Showcase) shown(p Product) bool {
	any := false
	for _, on := range s.categories {
		any = any || on
	}
	return !any || s.categories[p.Category]
}

// --- template data ---

func (s *Showcase) Tab() string { return s.tab }

// Tabs lists the tabs in order.
func (s *Showcase) Tabs() []struct{ ID, Label string } {
	return []struct{ ID, Label string }{{"order", "Order"}, {"calc", "Calculator"}, {"lang", "Languages"}, {"layout", "Layout"}}
}

// Products returns the products the category filters let through.
func (s *Showcase) Products() []Product {
	var out []Product
	for _, p := range s.products {
		if s.shown(p) {
			out = append(out, p)
		}
	}
	return out
}

// Categories lists the filter chips with their state.
func (s *Showcase) Categories() []struct {
	Name string
	On   bool
} {
	var out []struct {
		Name string
		On   bool
	}
	for _, c := range []string{"Fruit", "Bakery", "Dairy", "Pantry"} {
		out = append(out, struct {
			Name string
			On   bool
		}{c, s.categories[c]})
	}
	return out
}

// Selected returns the selected products (all categories).
func (s *Showcase) Selected() []Product {
	var out []Product
	for _, p := range s.products {
		if p.Selected {
			out = append(out, p)
		}
	}
	return out
}

// Count is the number of selected items (quantities added up).
func (s *Showcase) Count() int {
	n := 0
	for _, p := range s.Selected() {
		n += p.Qty
	}
	return n
}

func (s *Showcase) Subtotal() int {
	t := 0
	for _, p := range s.Selected() {
		t += p.Price * p.Qty
	}
	return t
}

// CouponOK reports whether the discount code is valid (SAVE10: 10% off).
func (s *Showcase) CouponOK() bool { return strings.EqualFold(strings.TrimSpace(s.coupon), "SAVE10") }

func (s *Showcase) CouponBad() bool { return strings.TrimSpace(s.coupon) != "" && !s.CouponOK() }

func (s *Showcase) Discount() int {
	if s.CouponOK() {
		return (s.Subtotal()*10 + 50) / 100
	}
	return 0
}

func (s *Showcase) Tax() int { return ((s.Subtotal()-s.Discount())*s.taxRate + 50) / 100 }

func (s *Showcase) Total() int { return s.Subtotal() - s.Discount() + s.Tax() }

func (s *Showcase) TaxRate() int      { return s.taxRate }
func (s *Showcase) Rates() []int      { return []int{0, 5, 18} }
func (s *Showcase) Coupon() string    { return s.coupon }
func (s *Showcase) Checkout() bool    { return s.checkout }
func (s *Showcase) Placed() string    { return s.placed }
func (s *Showcase) Display() string   { return s.display }
func (s *Showcase) PendingOp() string { return s.op }
func (s *Showcase) Typed() string     { return s.typed }

// Money formats cents for the template.
func (s *Showcase) Money(c int) string { return money(c) }

// LineTotal is a product's price times its quantity, formatted.
func (p Product) LineTotal() string { return money(p.Price * p.Qty) }

// PriceText is the unit price, formatted.
func (p Product) PriceText() string { return money(p.Price) }

func money(c int) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s$%d.%02d", sign, c/100, c%100)
}

// CalcKeys are the calculator's buttons, row by row.
func (s *Showcase) CalcKeys() [][]string {
	return [][]string{{"C", "⌫", "%", "÷"}, {"7", "8", "9", "×"}, {"4", "5", "6", "−"}, {"1", "2", "3", "+"}, {"0", ".", "="}}
}

// IsOp reports whether a calculator key is an operator (styled apart).
func (s *Showcase) IsOp(k string) bool { return strings.Contains("÷×−+=", k) }
