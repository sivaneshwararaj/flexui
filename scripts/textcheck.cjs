// Records Chromium's layout of the text documents in a directory: the
// border box of every element and anonymous text box (<x-t>) in tree
// order, and for every text box its lines (text with collapsible spaces
// trimmed, and the line's ink-independent rectangle from a Range).
// Prints JSON {name: {boxes: [[x,y,w,h]...], lines: [[[text,x,y,w,h]...]...]}}.
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');
const { load } = require('./flexui-page.cjs');

const dir = process.argv[2];

function measure() {
  const r = (e) => { const b = e.getBoundingClientRect(); return [b.x, b.y, b.width, b.height]; };
  const els = [document.body, ...document.body.querySelectorAll('*')].filter((e) => getComputedStyle(e).display !== 'none' || true);
  const boxes = els.filter((e) => !['STYLE', 'SCRIPT'].includes(e.tagName)).map((e) => {
    if (getComputedStyle(e).display === 'none' || e.closest('[style*="display: none"]')) return null;
    return r(e);
  });
  const lines = [];
  for (const t of document.body.querySelectorAll('x-t')) {
    const ws = getComputedStyle(t).whiteSpace;
    const pre = /^pre/.test(ws);
    // Characters with their line (by client rect top).
    const ls = [];
    const walker = document.createTreeWalker(t, NodeFilter.SHOW_TEXT);
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
      for (let i = 0; i < n.data.length; i++) {
        const rg = document.createRange();
        rg.setStart(n, i); rg.setEnd(n, i + 1);
        const rs = rg.getClientRects();
        if (!rs.length) continue; // collapsed white space
        const b = rs[0];
        let l = ls.find((l) => l.top === b.top);
        if (!l) { l = { top: b.top, text: '', x0: Infinity, x1: -Infinity, ranges: [] }; ls.push(l); }
        const ch = n.data[i];
        l.text += ch === '\n' ? '' : ch;
        if (b.width > 0 || !/\s/.test(ch)) {
          if (!/\s/.test(ch) || ws === 'pre') { l.x0 = Math.min(l.x0, b.left); l.x1 = Math.max(l.x1, b.right); }
        }
        l.h = b.height;
      }
    }
    lines.push(ls.map((l) => {
      let s = pre ? l.text.replace(/ +$/, '') : l.text.replace(/[ \t\n\r\f]+/g, ' ').trim();
      return [s, l.x0, l.top, l.x1 - l.x0, l.h];
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
