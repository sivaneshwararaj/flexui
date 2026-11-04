# Notes for coding agents

## If you are building an app WITH flexui (importing it)

Read [docs/GUIDE.md](docs/GUIDE.md) first; it is complete and checked
against the code. The essentials:

1. Module `github.com/sivaneshwararaj/flexui`, Go 1.24+, no cgo.
2. `app := flexui.New(flexui.Options{...})`, `app.SetViewFunc(view, css)`
   where `view()` returns HTML from your Go state (use `html/template`),
   `app.Handle("Name", fn)` for `onclick="Name"` etc. Handlers mutate
   state; flexui re-renders automatically.
3. Every element is a **flex column** container by default and
   `position: relative`. Use `flex-direction: row` for rows. No grid,
   floats, tables or block flow.
4. Inside a handler use only `Ctx` (`c.Value`, `c.InputValue(id)`,
   `c.Attr`, `c.Focus`, `c.StopPropagation`, `c.PreventDefault`). Calling
   locking `App` methods there deadlocks. `app.Invalidate()` is safe
   anywhere.
5. Give stable `id`s to inputs, scroll containers and focusable elements;
   focus, scroll and input text survive re-renders by id.
6. Validate markup and CSS with `flexui.CheckHTML(html, css)` and assert
   `app.Warnings()` is empty in tests: unsupported things are reported,
   never silently half-applied.
7. Test with `headless.Start(app)` + `s.Click/Type/Key/Wheel/...` and
   `app.Bounds(id)` to find elements. Frames are deterministic: compare
   golden PNGs byte for byte.
8. Windows: `windows.Run(app)` (only builds on windows); macOS:
   `darwin.Run(app)` (only builds on darwin). Put them behind build tags
   (see `examples/counter/window_*.go`).
9. Working, complete example: `examples/demo` (list + form + modal,
   scripted headless test with goldens).

Exact behavior of any element or property: [spec/SUBSET.md](spec/SUBSET.md).
Known gaps: [spec/STATUS.md](spec/STATUS.md).

## If you are changing flexui itself

Follow [claude.md](claude.md) (project rules) and read
[spec/ARCHITECTURE.md](spec/ARCHITECTURE.md) to orient yourself. Before
every commit: `make check` (gofmt, vet, tests in both modules, the
portability gate). Never regenerate goldens or loosen tolerances to make
a test pass; record deliberate differences from browsers as numbered
deviations in `spec/SUBSET.md`. Keep `spec/STATUS.md` honest.
