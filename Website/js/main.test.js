// SPDX-License-Identifier: Apache-2.0
//
// Anti-bluff smoke test for website/js/main.js.
//
// Loads the REAL script (no eval, no stub) via jsdom DOM injection, dispatches
// real DOM events, and asserts user-visible behaviour.  If main.js is replaced
// with an empty stub every assertion below fails — the test is not bluff.
//
// Self-check reasoning:
//   - assert(typeof window.trackEvent === 'function')  → fails on empty stub
//   - trackEvent() call must not throw                → fails if function missing
//   - scrollY>50 + scroll event → .navbar gets 'scrolled' → fails if listener absent

const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const { JSDOM } = require('jsdom');

test('main.js wires navigation and analytics for end users', () => {
    // Build a minimal DOM that satisfies all selectors used by main.js:
    //   .navbar  (initNavigation scroll handler)
    //   a[href^="#"]  (initSmoothScroll)
    //   div#x  (scroll target for the anchor)
    const dom = new JSDOM(
        `<!DOCTYPE html>
         <head><title>Test</title></head>
         <body>
           <nav class="navbar"></nav>
           <a href="#x">Go</a>
           <div id="x"></div>
         </body>`,
        {
            runScripts: 'dangerously',
            pretendToBeVisual: true,
            url: 'http://localhost/'
        }
    );

    // Load the REAL file via DOM injection — jsdom executes <script> text.
    const scriptPath = path.join(__dirname, 'main.js');
    const code = fs.readFileSync(scriptPath, 'utf8');
    const scriptEl = dom.window.document.createElement('script');
    scriptEl.textContent = code;
    dom.window.document.body.appendChild(scriptEl);

    // The IIFE in main.js registers a DOMContentLoaded listener.
    // window.trackEvent is assigned unconditionally (outside that listener),
    // so it is already available after script injection.
    // Fire DOMContentLoaded to trigger initNavigation() and register the scroll handler.
    dom.window.document.dispatchEvent(
        new dom.window.Event('DOMContentLoaded', { bubbles: true })
    );

    // --- Assertion 1: window.trackEvent is a function after load ---
    // An empty stub would not define this; fails if main.js body is absent.
    assert.strictEqual(
        typeof dom.window.trackEvent,
        'function',
        'window.trackEvent must be a function after main.js loads'
    );

    // --- Assertion 2: trackEvent() does not throw with valid arguments ---
    // Exercises the real function body (console.log path, optional gtag guard).
    assert.doesNotThrow(
        () => dom.window.trackEvent('test-category', 'test-action', 'test-label'),
        'trackEvent(cat, act, lbl) must not throw'
    );

    // --- Assertion 3: scroll > 50 causes .navbar to gain class "scrolled" ---
    // This verifies initNavigation() actually ran and registered the scroll handler.
    // Threshold in main.js: scrollY > 50  (i.e. 51 triggers it; 50 does not).
    Object.defineProperty(dom.window, 'scrollY', { value: 100, writable: true });
    dom.window.dispatchEvent(new dom.window.Event('scroll'));

    const navbar = dom.window.document.querySelector('.navbar');
    assert.ok(
        navbar !== null,
        '.navbar element must exist in the DOM'
    );
    assert.ok(
        navbar.classList.contains('scrolled'),
        '.navbar must have class "scrolled" after scrollY=100 + scroll event'
    );

    // --- Assertion 4: scrollY <= 50 removes the "scrolled" class (bi-directional) ---
    // Ensures the handler also removes the class on scroll-back — not just adds it.
    Object.defineProperty(dom.window, 'scrollY', { value: 0, writable: true });
    dom.window.dispatchEvent(new dom.window.Event('scroll'));
    assert.ok(
        !navbar.classList.contains('scrolled'),
        '.navbar must NOT have class "scrolled" after scrollY=0 + scroll event'
    );
});
