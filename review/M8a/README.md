# M8a review images

From `input_test.go` (synthetic pointer, key and text events in
`headless.Session`). `contact_sheet.png`, left to right, top to bottom:

1. `input_base.png`: placeholders, a disabled input with a value.
2. `input_typed.png`: after clicking "Name" and typing "Ada Lovelace"
   (controlled input: the echo below re-renders from state; caret kept).
3. `input_selected.png`: Ctrl+ArrowLeft then Shift+End selects
   "Lovelace".
4. `input_scrolled_end.png`: a long address typed into "Email": the text
   scrolls so the caret at the end is visible.
5. `input_scrolled_home.png`: after Home: scrolled back, caret at the
   start.
6. `input_drag_selected.png`: a pointer drag selected "a.very".
