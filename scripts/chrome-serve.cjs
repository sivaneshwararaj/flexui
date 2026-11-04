// Persistent Chromium layout oracle for the test-case reducer
// (go test ./internal/layout -run TestReduce -args -reduce=CASE).
// Reads one JSON-encoded HTML document per line on stdin and writes one
// JSON line {id: [x, y, w, h]} per document on stdout.
const readline = require('readline');
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 2000, height: 2000 } });
  const rl = readline.createInterface({ input: process.stdin });
  for await (const line of rl) {
    await page.setContent(JSON.parse(line));
    const r = await page.evaluate(() => {
      const vp = document.getElementById('vp').getBoundingClientRect();
      const r = {};
      for (const el of document.querySelectorAll('[data-id]')) {
        const b = el.getBoundingClientRect();
        r[el.dataset.id] = [b.x - vp.x, b.y - vp.y, b.width, b.height];
      }
      return r;
    });
    process.stdout.write(JSON.stringify(r) + '\n');
  }
  await browser.close();
})();
