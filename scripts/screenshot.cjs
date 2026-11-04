// Screenshots every document in a directory with headless Chromium, set up
// by flexui-page.cjs to follow flexui's subset. Writes
// <out>/<name>[@2x].png. Run via scripts/crosscheck.sh.
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');
const { load } = require('./flexui-page.cjs');

const dir = process.argv[2];
const out = process.argv[3];
const hidpi = new Set((process.argv[4] || '').split(',').filter(Boolean));

(async () => {
  const browser = await chromium.launch();
  fs.mkdirSync(out, { recursive: true });
  for (const f of fs.readdirSync(dir).filter((f) => f.endsWith('.html')).sort()) {
    const src = fs.readFileSync(path.join(dir, f), 'utf8');
    const name = f.slice(0, -5);
    for (const scale of hidpi.has(name) ? [1, 2] : [1]) {
      const { ctx, page } = await load(browser, src, scale);
      const file = path.join(out, name + (scale === 2 ? '@2x' : '') + '.png');
      await page.screenshot({ path: file, animations: 'disabled', caret: 'hide' });
      await ctx.close();
    }
  }
  await browser.close();
})();
