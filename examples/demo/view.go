package main

// view is the html/template of the demo; it is executed with *Demo.
const view = `<div class="app" onkeydown="Key">
<div class="sidebar">
  <div class="sidehead"><div class="brand">Contacts <span class="count">{{.Count}}</span></div>
    <button id="add" class="icon" onclick="Add">+ Add</button></div>
  <input id="search" class="search" placeholder="Search" value="{{.Query}}" oninput="Search">
  <div id="list" class="list">
  {{range .Visible}}<div class="item{{if eq .ID $.Selected}} selected{{end}}" onclick="Select" data-id="{{.ID}}">
    <div class="avatar">{{.Initials}}</div><div class="who"><div class="name">{{.Name}}</div><div class="mail">{{.Email}}</div></div></div>
  {{else}}<div class="empty">No contacts match “{{.Query}}”.</div>{{end}}
  </div>
</div>
<div class="main">
{{if .HasSelection}}
  <div class="title"><div class="avatar big">{{.Draft.Initials}}</div><div class="who"><div class="heading">{{.Draft.Name}}</div><div class="mail">{{.Draft.Email}}</div></div></div>
  <div class="form">
    <div class="field"><div class="label">Name</div><input id="name" value="{{.Draft.Name}}" oninput="EditName" placeholder="Full name"></div>
    <div class="field"><div class="label">Email</div><input id="email" value="{{.Draft.Email}}" oninput="EditEmail" placeholder="name@example.com"></div>
    <div class="field"><div class="label">Notes</div><input id="notes" value="{{.Draft.Notes}}" oninput="EditNotes" placeholder="Anything to remember"></div>
  </div>
  <div class="actions">
    <button id="delete" class="danger" onclick="AskDelete">Delete…</button>
    <div class="spacer"></div>
    <button id="revert" onclick="Revert" {{if not .Dirty}}disabled{{end}}>Revert</button>
    <button id="save" class="primary" onclick="Save" {{if not .Dirty}}disabled{{end}}>Save</button>
  </div>
  {{if .Status}}<div id="status" class="status">{{.Status}}</div>{{end}}
{{else}}
  <div class="none">Select a contact, or add one.</div>
{{end}}
</div>
{{if .Confirm}}
<div id="backdrop" class="backdrop" onclick="Backdrop">
  <div class="dialog">
    <div class="dtitle">Delete {{.Draft.Name}}?</div>
    <div class="dtext">This removes the contact and its notes. You cannot undo this.</div>
    <div class="dactions"><button id="cancel" onclick="CancelDelete">Cancel</button><button id="confirm" class="danger solid" onclick="ConfirmDelete">Delete</button></div>
  </div>
</div>
{{end}}
</div>`

const css = `
body { background-color: #f4f5f7; color: #1e222a; font-size: 13px; }
.app { flex-direction: row; flex-grow: 1; }
.sidebar { width: 250px; background-color: #fff; border-width: 0 1px 0 0; border-color: #e1e4e8; padding: 12px 0 0 0; gap: 8px; }
.sidehead { flex-direction: row; align-items: center; padding: 0 12px; }
.brand { font-size: 16px; font-weight: 700; flex-grow: 1; flex-direction: row; }
.count { color: #6b7280; font-size: 12px; font-weight: 400; }
.icon { padding: 3px 10px; }
.search { margin: 0 12px; width: auto; }
.list { overflow: scroll; flex-grow: 1; flex-basis: 0; border-width: 1px 0 0 0; border-color: #eef0f3; }
.item { flex-direction: row; align-items: center; gap: 10px; padding: 8px 12px; flex-shrink: 0; border-width: 0 0 1px 0; border-color: #f1f2f4; }
.item:hover { background-color: #f5f8ff; }
.item.selected { background-color: #e3ecff; }
.avatar { width: 30px; height: 30px; border-radius: 15px; background-color: #c7d7fe; color: #1e3a8a; font-weight: 700; font-size: 11px; align-items: center; justify-content: center; flex-shrink: 0; }
.avatar.big { width: 48px; height: 48px; border-radius: 24px; font-size: 17px; }
.who { gap: 1px; min-width: 0; flex-shrink: 1; }
.name { font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.mail { color: #6b7280; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.empty { color: #6b7280; padding: 16px 12px; }
.main { flex-grow: 1; padding: 24px; gap: 18px; }
.title { flex-direction: row; align-items: center; gap: 14px; }
.heading { font-size: 20px; font-weight: 700; }
.form { background-color: #fff; border: 1px solid #e1e4e8; border-radius: 8px; padding: 14px; gap: 10px; }
.field { flex-direction: row; align-items: center; gap: 12px; }
.label { width: 52px; color: #6b7280; }
.field input { flex-grow: 1; width: auto; }
.actions { flex-direction: row; gap: 8px; align-items: center; }
.spacer { flex-grow: 1; }
.primary { background-color: #3478f6; color: #fff; border-color: #2563eb; }
.primary:hover { background-color: #2563eb; }
.primary:disabled { background-color: #a9c4fb; border-color: #a9c4fb; color: #fff; }
.danger { color: #b42318; }
.danger.solid { background-color: #d92d20; color: #fff; border-color: #b42318; }
.status { color: #067647; }
.none { color: #6b7280; align-self: center; margin: auto; }
.backdrop { position: absolute; left: 0; top: 0; right: 0; bottom: 0; z-index: 10; background-color: rgba(15, 23, 42, 0.45); align-items: center; justify-content: center; }
.dialog { width: 340px; background-color: #fff; border-radius: 12px; padding: 20px; gap: 10px; box-shadow: 0 12px 32px rgba(0, 0, 0, 0.3); }
.dtitle { font-size: 16px; font-weight: 700; }
.dtext { color: #4b5563; line-height: 1.4; }
.dactions { flex-direction: row; justify-content: flex-end; gap: 8px; margin-top: 8px; }
`
