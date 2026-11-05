// Shared page setup for the Chromium oracle (screenshot.cjs, textcheck.cjs).
// It makes Chromium lay out a document the way flexui is specified to:
//  * standards mode (a doctype is added);
//  * base style sheet for flexui's defaults (spec/SUBSET.md D1-D4: every
//    element a flex column container, position relative, solid borders,
//    body filling the viewport, no UA styles, white canvas without
//    background propagation);
//  * the bundled Inter faces through @font-face (weights 1-500 Regular,
//    501-1000 Bold, as flexui's font matching), used by every element,
//    followed by the optional font packs (fonts/*) in flexui's fallback
//    order, so Chromium's per-cluster font fallback picks the same faces;
//  * flexui's default style sheet (internal/render/ua.css), loaded after
//    the base sheet so that document rules win;
//  * the lang attribute's content language kept (-webkit-locale);
//  * every maximal run of text and inline elements wrapped in an <x-t>
//    element equivalent to the anonymous block flexui creates for it
//    (spec/SUBSET.md §7.1), including the overflow / text-overflow it
//    takes from an ellipsis parent.
const fs = require('fs');
const path = require('path');

const fontDir = path.join(__dirname, '..', 'internal', 'text', 'fonts');
const uaCSS = fs.readFileSync(path.join(__dirname, '..', 'internal', 'render', 'ua.css'), 'utf8');
const b64 = (f) => fs.readFileSync(path.join(fontDir, f)).toString('base64');
// Optional packs in flexui's fallback order (their Go Priority).
const packDir = path.join(__dirname, '..', 'fonts');
// [dir, family, regular file, bold file (null: the regular face for all weights)]
const packs = [['tamil', 'NotoSansTamil', 'NotoSansTamil-Regular.ttf', 'NotoSansTamil-Bold.ttf'],
  ['devanagari', 'NotoSansDevanagari', 'NotoSansDevanagari-Regular.ttf', 'NotoSansDevanagari-Bold.ttf'],
  ['arabic', 'NotoSansArabic', 'NotoSansArabic-Regular.ttf', 'NotoSansArabic-Bold.ttf'],
  ['hebrew', 'NotoSansHebrew', 'NotoSansHebrew-Regular.ttf', 'NotoSansHebrew-Bold.ttf'],
  ['cjk', 'NotoSansSC', 'NotoSansSC-Regular.otf', null]];
const pb64 = (dir, f) => fs.readFileSync(path.join(packDir, dir, f)).toString('base64');
const fontFaces = `@font-face { font-family: FlexUI; font-weight: 1 500;
  src: url(data:font/ttf;base64,${b64('Inter-Regular.ttf')}); }
@font-face { font-family: FlexUI; font-weight: 501 1000;
  src: url(data:font/ttf;base64,${b64('Inter-Bold.ttf')}); }
` + packs.map(([dir, name, reg, bold]) => bold ? `@font-face { font-family: ${name}; font-weight: 1 500;
  src: url(data:font/ttf;base64,${pb64(dir, reg)}); }
@font-face { font-family: ${name}; font-weight: 501 1000;
  src: url(data:font/ttf;base64,${pb64(dir, bold)}); }` : `@font-face { font-family: ${name}; font-weight: 1 1000;
  src: url(data:font/otf;base64,${pb64(dir, reg)}); }`).join('\n');
const families = ['FlexUI', ...packs.map(([, name]) => name)].join(', ');

const base = (w, h) => `<style>
${fontFaces}
html { margin: 0; padding: 0; background: #fff; display: grid;
  grid-template-columns: minmax(0, 1fr); grid-template-rows: minmax(0, 1fr);
  width: ${w}px; height: ${h}px; overflow: hidden; font-size: 16px; }
:where(body, body *) { all: unset; display: flex; flex-direction: column;
  position: relative; box-sizing: content-box; min-width: auto; min-height: auto;
  border: 0 solid currentcolor; }
body { font-family: ${families}; font-size: 16px; font-weight: 400; line-height: normal; color: #000; }
* { font-family: ${families} !important; font-kerning: normal; font-synthesis: none;
  -webkit-font-smoothing: antialiased; }
x-t { all: unset; display: block; font-family: ${families} !important; }
</style><style>
/* flexui's default style sheet (internal/render/ua.css) */
${uaCSS}
</style>`;

// Runs in the page: wraps text runs, then waits for fonts.
function wrapText() {
  const skip = new Set(['STYLE', 'SCRIPT', 'TITLE', 'X-T']);
  // An element is inline if its display is inline (Chromium reports
  // "block" for inline children of flex containers: blockified) and all
  // its descendant elements are inline too.
  const isInline = (e) => {
    const d = getComputedStyle(e).display;
    if (d === 'flex' || d === 'none' || skip.has(e.tagName)) return false;
    return [...e.children].every(isInline);
  };
  const walk = (el) => {
    let run = [];
    const flush = () => {
      if (run.length && run.some((n) => /[^ \t\n\r\f]/.test(n.textContent))) {
        const w = document.createElement('x-t');
        const cs = getComputedStyle(el);
        if (cs.textOverflow === 'ellipsis' && cs.overflowX !== 'visible') {
          w.style.overflow = cs.overflowX;
          w.style.textOverflow = 'ellipsis';
        }
        run[0].before(w);
        for (const n of run) w.appendChild(n);
      }
      run = [];
    };
    for (const n of [...el.childNodes]) {
      if (n.nodeType === Node.TEXT_NODE || (n.nodeType === Node.ELEMENT_NODE && isInline(n))) {
        run.push(n);
      } else if (n.nodeType === Node.COMMENT_NODE) {
        continue;
      } else {
        flush();
        if (n.nodeType === Node.ELEMENT_NODE && !skip.has(n.tagName)) walk(n);
      }
    }
    flush();
  };
  walk(document.body);
  // `all: unset` in the base sheet also resets the content language an
  // element takes from its own lang attribute (Chromium keeps it in the
  // internal -webkit-locale property): restore it.
  for (const e of document.querySelectorAll('[lang]')) {
    e.style.setProperty('-webkit-locale', JSON.stringify(e.getAttribute('lang')));
  }
  return document.fonts.ready.then(() => document.fonts.size);
}

async function load(browser, src, scale) {
  const m = /^<!-- size: (\d+)x(\d+) -->/.exec(src);
  const [w, h] = [Number(m[1]), Number(m[2])];
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: scale });
  const page = await ctx.newPage();
  // Base sheet first, so the document's rules win on equal specificity.
  // Standards mode (flexui has no quirks): without a doctype Chromium
  // would apply quirks such as the line-height calculation quirk.
  const doc = src.replace('<head>', '<head>' + base(w, h));
  await page.setContent(/^\s*(<!--[^]*?-->\s*)?<!doctype/i.test(doc) ? doc : '<!DOCTYPE html>' + doc);
  await page.evaluate(wrapText);
  await page.evaluate(() => document.fonts.ready);
  await page.mouse.move(-10, -10);
  return { ctx, page, w, h };
}

module.exports = { load };
