// Loads every HTML file in a directory in headless Chromium and prints
// {caseName: {id: [x, y, w, h]}} for all elements with data-id, relative
// to #vp. Used by scripts/crosscheck.sh; requires node + playwright.
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

(async () => {
  const dir = process.argv[2];
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 2000, height: 2000 } });
  const out = {};
  for (const f of fs.readdirSync(dir).filter((f) => f.endsWith('.html')).sort()) {
    await page.goto('file://' + path.resolve(dir, f));
    out[f.slice(0, -5)] = await page.evaluate(() => {
      const vp = document.getElementById('vp').getBoundingClientRect();
      const r = {};
      for (const el of document.querySelectorAll('[data-id]')) {
        const b = el.getBoundingClientRect();
        r[el.dataset.id] = [b.x - vp.x, b.y - vp.y, b.width, b.height];
      }
      return r;
    });
  }
  await browser.close();
  // Stable output: sorted keys, one case per line.
  const lines = Object.keys(out).sort().map((k) => {
    const ids = Object.keys(out[k]).sort();
    const body = ids.map((id) => JSON.stringify(id) + ':' + JSON.stringify(out[k][id])).join(',');
    return '  ' + JSON.stringify(k) + ': {' + body + '}';
  });
  process.stdout.write('{\n' + lines.join(',\n') + '\n}\n');
})();
