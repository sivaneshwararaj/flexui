// Screenshots every document in testdata/render/docs with headless
// Chromium, using a base style sheet that encodes flexui's defaults
// (spec/SUBSET.md D1-D4: every element a flex column container, position
// relative, solid borders, body filling the viewport, no UA styles, white
// canvas without background propagation). Writes
// testdata/render/chrome/<name>[@2x].png. Run via scripts/crosscheck.sh.
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const dir = process.argv[2];
const out = process.argv[3];
const hidpi = new Set((process.argv[4] || '').split(',').filter(Boolean));

const base = (w, h) => `<style>
html { margin: 0; padding: 0; background: #fff; display: grid;
  grid-template-columns: minmax(0, 1fr); grid-template-rows: minmax(0, 1fr);
  width: ${w}px; height: ${h}px; overflow: hidden; }
:where(body, body *) { all: unset; display: flex; flex-direction: column;
  position: relative; box-sizing: content-box; min-width: auto; min-height: auto;
  border: 0 solid currentcolor; }
</style>`;

(async () => {
  const browser = await chromium.launch();
  fs.mkdirSync(out, { recursive: true });
  for (const f of fs.readdirSync(dir).filter((f) => f.endsWith('.html')).sort()) {
    const src = fs.readFileSync(path.join(dir, f), 'utf8');
    const m = /^<!-- size: (\d+)x(\d+) -->/.exec(src);
    const [w, h] = [Number(m[1]), Number(m[2])];
    const name = f.slice(0, -5);
    const scales = hidpi.has(name) ? [1, 2] : [1];
    for (const scale of scales) {
      const ctx = await browser.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: scale });
      const page = await ctx.newPage();
      // Base sheet first, so the document's rules win on equal specificity.
      await page.setContent(src.replace('<head>', '<head>' + base(w, h)));
      await page.mouse.move(-10, -10);
      const file = path.join(out, name + (scale === 2 ? '@2x' : '') + '.png');
      await page.screenshot({ path: file, animations: 'disabled', caret: 'hide' });
      await ctx.close();
    }
  }
  await browser.close();
})();
