import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (path: string) => readFileSync(new URL(path, import.meta.url), "utf8");
const tokens = read("../app/design-tokens.css");
const css = read("../app/globals.css");
const definitions = new Map([...tokens.matchAll(/(--pf-[\w-]+):\s*([^;]+);/g)].map(m => [m[1], m[2]]));
function resolve(name: string): string {
  const value = definitions.get(name);
  assert.ok(value, `missing token: ${name}`);
  const alias = /^var\((--pf-[\w-]+)\)$/.exec(value);
  return alias ? resolve(alias[1]) : value;
}
function luminance(hex: string) {
  const rgb = hex.replace("#", "").match(/../g)!.map(v => parseInt(v, 16) / 255)
    .map(v => v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4);
  return rgb[0] * 0.2126 + rgb[1] * 0.7152 + rgb[2] * 0.0722;
}
test("approved palette is imported and every pf reference resolves", () => {
  assert.match(css, /@import "\.\/design-tokens.css"/);
  assert.equal(resolve("--pf-primary-500"), "#2F73FF");
  assert.equal(resolve("--pf-bg-app"), "#F7FAFD");
  for (const match of (css + tokens).matchAll(/var\((--pf-[\w-]+)/g)) assert.ok(definitions.has(match[1]), match[1]);
  assert.match(read("../../AGENTS.md"), /docs\/design-system.md/);
});
test("readable text and primary button aliases meet AA contrast", () => {
  for (const [fg, bg] of [["--pf-text-primary", "--pf-bg-app"], ["--pf-text-readable-muted", "--pf-surface"], ["--pf-text-inverse", "--pf-button-bg"], ["--pf-text-inverse", "--pf-button-hover"]]) {
    const values = [luminance(resolve(fg)), luminance(resolve(bg))].sort((a,b) => b-a);
    assert.ok((values[0]+0.05)/(values[1]+0.05) >= 4.5, `${fg} on ${bg}`);
  }
});
