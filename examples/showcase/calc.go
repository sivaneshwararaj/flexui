package main

import (
	"strconv"
	"strings"
)

// press handles one calculator key: digits and "." build the number on
// the display, operators apply the pending one (left to right, like a
// pocket calculator), "=" finishes, "%" divides by 100, "⌫" deletes a
// digit, "C" clears.
func (s *Showcase) press(k string) {
	switch {
	case k == "C":
		s.display, s.acc, s.op, s.fresh = "0", 0, "", true
	case k == "⌫":
		if s.fresh {
			return
		}
		s.display = s.display[:len(s.display)-1]
		if s.display == "" || s.display == "-" {
			s.display, s.fresh = "0", true
		}
	case k == "%":
		s.display, s.fresh = format(s.value()/100), true
	case k == ".":
		if s.fresh {
			s.display, s.fresh = "0.", false
		} else if !strings.Contains(s.display, ".") {
			s.display += "."
		}
	case k >= "0" && k <= "9":
		if s.fresh || s.display == "0" {
			s.display, s.fresh = k, false
		} else if len(s.display) < 15 {
			s.display += k
		}
	default: // + − × ÷ =
		if s.op != "" && !s.fresh {
			s.display = format(apply(s.acc, s.value(), s.op))
		}
		s.acc, s.fresh = s.value(), true
		s.op = k
		if k == "=" {
			s.op = ""
		}
	}
}

func (s *Showcase) value() float64 {
	v, _ := strconv.ParseFloat(s.display, 64)
	return v
}

func apply(a, b float64, op string) float64 {
	switch op {
	case "+":
		return a + b
	case "−":
		return a - b
	case "×":
		return a * b
	case "÷":
		if b == 0 {
			return 0
		}
		return a / b
	}
	return b
}

// format prints a result with at most 10 significant digits and no
// trailing zeros.
func format(v float64) string {
	return strconv.FormatFloat(v, 'g', 10, 64)
}
