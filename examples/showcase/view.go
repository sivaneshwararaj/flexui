package main

// view is the html/template of the showcase; it is executed with
// *Showcase.
const view = `<div class="app" onkeydown="Key">
<div class="top">
  <div class="brand">flexui <span class="sub">showcase</span></div>
  <div class="tabs">{{range .Tabs}}<button id="tab-{{.ID}}" class="tab{{if eq .ID $.Tab}} on{{end}}" onclick="Tab" data-tab="{{.ID}}">{{.Label}}</button>{{end}}</div>
</div>
<div class="page">
{{if eq .Tab "order"}}
  <div class="order">
    <div class="panel products">
      <div class="bar">
        <div class="chips">{{range .Categories}}<button id="cat-{{.Name}}" class="chip{{if .On}} on{{end}}" onclick="Filter" data-cat="{{.Name}}">{{.Name}}</button>{{end}}</div>
        <div class="spacer"></div>
        <button id="all" onclick="SelectAll">Select all</button>
        <button id="clear" onclick="Clear">Clear</button>
      </div>
      <div id="list" class="list">
      {{range .Products}}<div id="p{{.ID}}" class="row{{if .Selected}} sel{{end}}" onclick="Toggle" data-id="{{.ID}}">
        <div class="check{{if .Selected}} on{{end}}"><div class="tick"></div></div>
        <div class="pname"><div class="name">{{.Name}}</div><div class="cat">{{.Category}} · {{.PriceText}}</div></div>
        <div class="qty"><button id="dec{{.ID}}" class="step" onclick="Dec" data-id="{{.ID}}">−</button><div class="n">{{.Qty}}</div><button id="inc{{.ID}}" class="step" onclick="Inc" data-id="{{.ID}}">+</button></div>
        <div class="line">{{.LineTotal}}</div>
      </div>{{end}}
      </div>
    </div>
    <div class="panel summary">
      <div class="h2">Summary</div>
      <div class="kv"><div>Items</div><div id="count">{{.Count}}</div></div>
      <div class="kv"><div>Subtotal</div><div id="subtotal">{{.Money .Subtotal}}</div></div>
      <div class="label">Tax rate</div>
      <div class="seg">{{range .Rates}}<button id="tax{{.}}" class="segb{{if eq . $.TaxRate}} on{{end}}" onclick="Tax" data-rate="{{.}}">{{.}}%</button>{{end}}</div>
      <div class="label">Discount code</div>
      <input id="coupon" value="{{.Coupon}}" placeholder="try SAVE10" oninput="Coupon">
      {{if .CouponOK}}<div class="ok">10% off applied</div>{{else if .CouponBad}}<div class="bad">Unknown code</div>{{end}}
      <div class="kv"><div>Discount</div><div id="discount">-{{.Money .Discount}}</div></div>
      <div class="kv"><div>Tax</div><div id="tax">{{.Money .Tax}}</div></div>
      <div class="kv total"><div>Total</div><div id="total">{{.Money .Total}}</div></div>
      <button id="checkout" class="primary" onclick="Checkout" {{if eq .Count 0}}disabled{{end}}>Checkout</button>
      {{if .Placed}}<div id="placed" class="ok">{{.Placed}}</div>{{end}}
    </div>
  </div>
{{else if eq .Tab "calc"}}
  <div class="calcwrap">
    <div id="calc" class="calc" tabindex="0" onkeydown="CalcKey">
      <div class="disp"><div class="pending">{{.PendingOp}}</div><div id="display" class="dval">{{.Display}}</div></div>
      {{range .CalcKeys}}<div class="krow">{{range .}}<button id="k{{.}}" class="key{{if $.IsOp .}} op{{end}}{{if eq . "0"}} wide{{end}}" onclick="CalcButton" data-k="{{.}}">{{.}}</button>{{end}}</div>{{end}}
      <div class="hint">Click the keys, or click here and type: 0-9 . + - * / Enter Backspace Escape</div>
    </div>
  </div>
{{else if eq .Tab "lang"}}
  <div class="langs">
    <div class="cards">
      <div class="card" lang="ta"><div class="lname">Tamil</div><div class="greet">வணக்கம் உலகம்</div><div class="small">தமிழ் உலகின் பழமையான மொழிகளில் ஒன்று.</div></div>
      <div class="card" lang="hi"><div class="lname">Hindi</div><div class="greet">नमस्ते दुनिया</div><div class="small">हिन्दी देवनागरी लिपि में लिखी जाती है।</div></div>
      <div class="card" dir="rtl" lang="ar"><div class="lname">Arabic</div><div class="greet">مرحبا بالعالم</div><div class="small">تكتب العربية من اليمين إلى اليسار، والأرقام ١٢٣ من اليسار.</div></div>
      <div class="card" dir="rtl" lang="he"><div class="lname">Hebrew</div><div class="greet">שלום עולם</div><div class="small">עברית עם English ומספרים 2024 באותה שורה.</div></div>
      <div class="card"><div class="lname">Chinese</div><div class="greet">你好，世界</div><div class="small">中文是世界上使用人数最多的语言之一。</div></div>
      <div class="card" lang="ja"><div class="lname">Japanese</div><div class="greet">こんにちは世界</div><div class="small">ひらがなとカタカナと漢字</div></div>
    </div>
    <div class="panel tryit">
      <div class="label">Type or paste in any language (bidi, clusters and IME work here)</div>
      <input id="typed" value="{{.Typed}}" placeholder="Type here…" oninput="Typed">
      <div class="echo">{{if .Typed}}<span dir="auto">{{.Typed}}</span>{{else}}Your text appears here.{{end}}</div>
    </div>
  </div>
{{else}}
  <div class="layout">
    <div class="panel"><div class="h2">justify-content</div>
      {{range $j := (list "flex-start" "center" "flex-end" "space-between" "space-around")}}<div class="jl">{{$j}}</div><div class="jrow" style="justify-content: {{$j}}"><div class="dot"></div><div class="dot"></div><div class="dot"></div></div>{{end}}
    </div>
    <div class="panel"><div class="h2">flex-wrap, gap, radius</div>
      <div class="wrap">{{range (list "Go" "HTML" "CSS" "flexbox" "no cgo" "deterministic" "headless tests" "HiDPI" "bidi" "IME" "font packs" "goldens")}}<div class="tag">{{.}}</div>{{end}}</div>
    </div>
    <div class="panel"><div class="h2">shadow, opacity, z-index</div>
      <div class="stack"><div class="sq a">1</div><div class="sq b">2</div><div class="sq c">3</div></div>
    </div>
    <div class="panel"><div class="h2">overflow: scroll</div>
      <div id="scroller" class="scroller">{{range (list 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20)}}<div class="srow">Row {{.}}</div>{{end}}</div>
    </div>
  </div>
{{end}}
</div>
{{if .Checkout}}
<div id="backdrop" class="backdrop" onclick="Backdrop">
  <div class="dialog">
    <div class="h2">Confirm your order</div>
    <div id="lines" class="dlines">{{range .Selected}}<div class="kv"><div>{{.Qty}} × {{.Name}}</div><div>{{.LineTotal}}</div></div>{{end}}</div>
    <div class="kv total"><div>Total</div><div>{{.Money .Total}}</div></div>
    <div class="dactions"><button id="cancel" onclick="Cancel">Cancel</button><button id="place" class="primary" onclick="Place">Place order</button></div>
  </div>
</div>
{{end}}
</div>`

const css = `
body { background-color: #f1f3f7; color: #1e222a; font-size: 13px; }
.app { flex-grow: 1; }
.top { flex-direction: row; align-items: center; gap: 24px; padding: 10px 16px; background-color: #1e293b; }
.brand { color: #fff; font-size: 17px; font-weight: 700; flex-direction: row; }
.sub { color: #94a3b8; font-weight: 400; }
.tabs { flex-direction: row; gap: 4px; }
.tab { background-color: transparent; border-color: transparent; color: #cbd5e1; padding: 5px 12px; border-radius: 6px; }
.tab:hover { background-color: #334155; }
.tab.on { background-color: #3b82f6; color: #fff; }
.page { flex-grow: 1; flex-basis: 0; padding: 14px; }
.panel { background-color: #fff; border: 1px solid #e2e6ec; border-radius: 10px; padding: 12px; gap: 8px; box-shadow: 0 1px 2px rgba(15, 23, 42, 0.06); }
.h2 { font-size: 15px; font-weight: 700; }
.label { color: #64748b; font-size: 12px; }
.spacer { flex-grow: 1; }
.primary { background-color: #3478f6; color: #fff; border-color: #2563eb; }
.primary:hover { background-color: #2563eb; }
.primary:disabled { background-color: #a9c4fb; border-color: #a9c4fb; }
.ok { color: #067647; }
.bad { color: #b42318; }

.order { flex-direction: row; gap: 14px; flex-grow: 1; }
.products { flex-grow: 1; flex-basis: 0; padding: 0; gap: 0; }
.bar { flex-direction: row; align-items: center; gap: 6px; padding: 10px 12px; border-width: 0 0 1px 0; border-color: #eef0f3; }
.chips { flex-direction: row; gap: 6px; }
.chip { border-radius: 14px; padding: 3px 10px; }
.chip.on { background-color: #dbe7ff; border-color: #93b4f8; color: #1d4ed8; }
.list { overflow: scroll; flex-grow: 1; flex-basis: 0; }
.row { flex-direction: row; align-items: center; gap: 10px; padding: 7px 12px; flex-shrink: 0; border-width: 0 0 1px 0; border-color: #f2f4f7; }
.row:hover { background-color: #f6f9ff; }
.row.sel { background-color: #eef4ff; }
.check { width: 16px; height: 16px; border: 1px solid #94a3b8; border-radius: 4px; align-items: center; justify-content: center; flex-shrink: 0; background-color: #fff; }
.check.on { background-color: #3478f6; border-color: #2563eb; }
.tick { width: 6px; height: 6px; border-radius: 1px; background-color: #fff; }
.pname { flex-grow: 1; min-width: 0; gap: 1px; }
.name { font-weight: 700; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cat { color: #64748b; font-size: 12px; }
.qty { flex-direction: row; align-items: center; gap: 6px; }
.step { width: 24px; padding: 2px 0; }
.n { width: 18px; align-items: center; }
.line { width: 64px; align-items: flex-end; font-weight: 700; }
.summary { width: 230px; }
.kv { flex-direction: row; justify-content: space-between; }
.total { font-size: 16px; font-weight: 700; border-width: 1px 0 0 0; border-color: #e2e6ec; padding: 8px 0 0 0; }
.seg { flex-direction: row; }
.segb { flex-grow: 1; border-radius: 0; }
.segb.on { background-color: #1e293b; color: #fff; border-color: #1e293b; }
.summary input { width: auto; }

.calcwrap { align-items: center; justify-content: center; flex-grow: 1; }
.calc { width: 260px; background-color: #1e293b; border-radius: 16px; padding: 14px; gap: 8px; box-shadow: 0 10px 24px rgba(15, 23, 42, 0.3); }
.calc:focus { border: 2px solid #60a5fa; padding: 12px; }
.disp { background-color: #0f172a; border-radius: 10px; padding: 10px 12px; align-items: flex-end; gap: 2px; }
.pending { color: #60a5fa; font-size: 12px; height: 15px; }
.dval { color: #fff; font-size: 30px; white-space: nowrap; }
.krow { flex-direction: row; gap: 8px; }
.key { flex-grow: 1; flex-basis: 0; height: 40px; font-size: 17px; background-color: #334155; color: #fff; border-color: #334155; border-radius: 10px; }
.key:hover { background-color: #475569; }
.key:active { background-color: #64748b; }
.key.op { background-color: #f59e0b; border-color: #f59e0b; }
.key.op:hover { background-color: #fbbf24; }
.key.wide { flex-grow: 2.1; }
.hint { color: #94a3b8; font-size: 11px; }

.langs { gap: 12px; flex-grow: 1; }
.cards { flex-direction: row; flex-wrap: wrap; gap: 10px; }
.card { width: 252px; background-color: #fff; border: 1px solid #e2e6ec; border-radius: 10px; padding: 10px 12px; gap: 4px; }
.lname { color: #64748b; font-size: 11px; font-weight: 700; }
.greet { font-size: 20px; font-weight: 700; }
.small { color: #334155; }
.tryit input { width: auto; }
.echo { font-size: 18px; color: #1d4ed8; flex-direction: row; }

.layout { flex-direction: row; flex-wrap: wrap; gap: 12px; align-content: flex-start; }
.layout .panel { width: 370px; }
.jrow { flex-direction: row; align-items: center; gap: 6px; background-color: #f6f8fb; border-radius: 6px; padding: 3px; }
.jl { color: #64748b; font-size: 11px; }
.dot { width: 16px; height: 16px; border-radius: 8px; background-color: #3478f6; }
.wrap { flex-direction: row; flex-wrap: wrap; gap: 6px; }
.tag { background-color: #eef2ff; color: #3730a3; border-radius: 12px; padding: 3px 10px; }
.stack { height: 92px; }
.sq { position: absolute; width: 70px; height: 70px; border-radius: 12px; align-items: center; justify-content: center; color: #fff; font-size: 20px; font-weight: 700; box-shadow: 0 6px 14px rgba(15, 23, 42, 0.25); }
.sq.a { left: 10px; top: 4px; background-color: #ef4444; z-index: 3; }
.sq.b { left: 60px; top: 14px; background-color: #22c55e; z-index: 2; opacity: 0.8; }
.sq.c { left: 110px; top: 22px; background-color: #3b82f6; z-index: 1; opacity: 0.6; }
.scroller { height: 90px; overflow: scroll; border: 1px solid #e2e6ec; border-radius: 6px; }
.srow { padding: 4px 8px; flex-shrink: 0; border-width: 0 0 1px 0; border-color: #f2f4f7; }

.backdrop { position: absolute; left: 0; top: 0; right: 0; bottom: 0; z-index: 10; background-color: rgba(15, 23, 42, 0.45); align-items: center; justify-content: center; }
.dialog { width: 340px; background-color: #fff; border-radius: 12px; padding: 18px; gap: 10px; box-shadow: 0 20px 40px rgba(15, 23, 42, 0.35); }
.dlines { gap: 4px; }
.dactions { flex-direction: row; justify-content: flex-end; gap: 8px; }
`
