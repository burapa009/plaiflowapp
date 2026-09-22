import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const css = readFileSync(new URL("../app/globals.css", import.meta.url), "utf8");
const navigation = readFileSync(new URL("../app/nav-links.tsx", import.meta.url), "utf8");

test("every app route uses the shared Tailwind brand shell", () => {
  assert.match(css, /\.shell\s*\{\s*@apply[^}]*bg-canvas[^}]*text-fg/);
  assert.doesNotMatch(css, /\.shell:has\(\.workspace-page\)/);
  assert.doesNotMatch(css, /#(?:f8fffa|4c6e5d|dff2e6|258865|175846|365d4d)/i);
  assert.match(css, /\.workspace-page \.work-header h1[^}]*linear-gradient/);
  assert.match(navigation, /className="nav-icon" aria-hidden="true"/);
});
