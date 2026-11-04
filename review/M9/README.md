# M9 review images

The demo app (`examples/demo`, public API only) driven by its scripted
headless session (`script.go`); these are its goldens. In order:

1. `1-start.png`: 14 contacts, the first selected, list scrollbar.
2. `2-search.png`: typing "ar" in the search field filters the list.
3. `3-selected.png`: clicking Margaret Hamilton fills the form.
4. `4-edited.png`: the email is edited (select all + type); Revert and
   Save are enabled because the form is dirty.
5. `5-saved.png`: saved; the list shows the new email; status line.
6. `6-dialog.png`: Delete… opens a modal over a dimmed backdrop.
7. `7-deleted.png`: after Escape (closes), Delete… and Delete: the
   contact is gone and the next visible one is selected.
8. `8-added.png`: search cleared, "+ Add", typed a name and email, saved.

`contact_sheet.png` shows all eight.
