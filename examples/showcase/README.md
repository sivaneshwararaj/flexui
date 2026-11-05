# flexui showcase

A tour of what flexui can do, written against the public API only
(`app.go` state and handlers, `view.go` HTML template and CSS):

| Tab | Shows |
|---|---|
| **Order** | Multi-select list (click rows), category filter chips (multi-select), quantity steppers, Select all / Clear, a segmented tax control, a discount code input (`SAVE10`), live totals in integer cents, a checkout dialog (modal over a backdrop, Escape or a click outside closes it), a scroll container. |
| **Calculator** | A button grid with hover/active styles, and keyboard input (`onkeydown` on a focusable element: digits, `+ - * /`, Enter, Backspace, Escape). |
| **Languages** | Tamil, Hindi, Arabic and Hebrew (right to left), Chinese and Japanese with the optional font packs; a field to type or paste any language (bidi, clusters, IME). |
| **Layout** | `justify-content`, `flex-wrap` with `gap`, rounded corners, box shadows, opacity, `z-index`, `overflow: scroll`. |

```sh
cd examples
go run ./showcase            # headless: writes showcase-*.png of a scripted session
go run ./showcase -window    # native window on macOS or Windows
go test ./showcase           # the scripted session against golden PNGs, plus value checks
```

The font packs are imported in `main.go`; without them the Languages tab
would draw boxes (and `app.Warnings()` would say which pack is missing).
