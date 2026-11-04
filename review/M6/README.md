# M6 review images

Produced by `interact_test.go` with synthetic events (`headless.Session`)
on a toolbar + scrollable list app. `contact_sheet.png` shows them in
order (left to right, top to bottom):

1. `list_base.png`: initial frame.
2. `list_hover_add.png`: pointer over "Add" (`:hover` background).
3. `list_active_clear.png`: "Clear" pressed (`:active`).
4. `list_focus_add.png`: after Tab x4 (wraps to "Add"; `:focus` border).
5. `list_focus_after_add.png`: Enter on the focused "Add" added an item;
   the view was re-rendered and focus was kept.
6. `list_scroll_100.png`: list scrolled 100px with the wheel (content
   clipped by the rounded list box).
7. `list_scrolled_selected.png`: after a drag scroll of 58px, a click
   selected the row under the pointer ("Item 03"); the re-render kept the
   scroll position. The status text wraps, making the toolbar taller.
