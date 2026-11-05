// Records Chromium's layout of the text documents in a directory: the
// border box of every element and anonymous text box (<x-t>) in tree
// order (inline elements: the union of their fragments), and for every
// text box its lines (text with collapsible spaces trimmed, and the
// extent of its characters' content areas).
// Prints JSON {name: {boxes: [[x,y,w,h]...], lines: [[[text,x,y,w,h]...]...]}}.
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');
const { load } = require('./flexui-page.cjs');

const dir = process.argv[2];

function measure() {
  // A document with right-to-left text or a dir attribute.
  const bidiDoc = /[\u0590-\u08ff\ufb1d-\ufdff\ufe70-\ufeff]/.test(document.body.textContent) ||
    !!document.querySelector('[dir]');
  const r = (e) => { const b = e.getBoundingClientRect(); return [b.x, b.y, b.width, b.height]; };
  const els = [document.body, ...document.body.querySelectorAll('*')]
    .filter((e) => !['STYLE', 'SCRIPT', 'BR'].includes(e.tagName));
  const boxes = els.map(r);
  const lines = [];
  for (const t of document.body.querySelectorAll('x-t')) {
    const ws = getComputedStyle(t).whiteSpace;
    const pre = /^pre/.test(ws);
    // Characters in order; a character starts a new line when it is
    // entirely below the current one (lines never overlap vertically).
    const ls = [];
    let cur = null;
    const walker = document.createTreeWalker(t, NodeFilter.SHOW_TEXT);
    // Grapheme clusters, not code units: a range inside a cluster (a
    // vowel sign, a virama) has no box of its own.
    const graphemes = new Intl.Segmenter(undefined, { granularity: 'grapheme' });
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
      for (const { segment: ch, index: i } of graphemes.segment(n.data)) {
        const rg = document.createRange();
        rg.setStart(n, i); rg.setEnd(n, i + ch.length);
        const rs = rg.getClientRects();
        if (!rs.length) continue; // collapsed white space
        const b = rs[0];
        if (b.width === 0 && /\s/.test(ch)) continue; // collapsed space
        // New line: entirely below the current one, or the pen moved back
        // left: character left edges increase along a line, even when kerned
        // (lines can overlap vertically with a small line-height).
        // In bidi text the pen moves both ways: only "entirely below"
        // starts a line (bidi documents avoid overlapping lines).
        if (!cur || b.top >= cur.bottom || (!bidiDoc && b.left <= cur.lastLeft)) {
          cur = { top: b.top, bottom: b.bottom, text: '', x0: Infinity, x1: -Infinity, lastLeft: -Infinity };
          ls.push(cur);
        }
        cur.lastLeft = b.left;
        cur.top = Math.min(cur.top, b.top);
        cur.bottom = Math.max(cur.bottom, b.bottom);
        cur.text += ch === '\n' ? '' : ch;
        if (!/\s/.test(ch) || ws === 'pre') { cur.x0 = Math.min(cur.x0, b.left); cur.x1 = Math.max(cur.x1, b.right); }
      }
    }
    lines.push(ls.map((l) => {
      const s = pre ? l.text.replace(/ +$/, '') : l.text.replace(/[ \t\n\r\f]+/g, ' ').trim();
      return [s, l.x0, l.top, l.x1 - l.x0, l.bottom - l.top];
    }));
  }
  return { boxes, lines };
}

(async () => {
  const browser = await chromium.launch();
  const out = {};
  for (const f of fs.readdirSync(dir).filter((f) => f.endsWith('.html')).sort()) {
    const src = fs.readFileSync(path.join(dir, f), 'utf8');
    const { ctx, page } = await load(browser, src, 1);
    out[f.slice(0, -5)] = await page.evaluate(measure);
    await ctx.close();
  }
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})();
