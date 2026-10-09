"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const grammar = JSON.parse(fs.readFileSync(
  path.join(__dirname, "..", "..", "syntaxes", "tgo.tmLanguage.json"),
  "utf8"
));
function pattern(name) {
  const patterns = [
    ...grammar.repository.tgo.patterns,
    ...Object.values(grammar.injections).flatMap((value) => value.patterns)
  ];
  const item = patterns.find((value) => value.name === name);
  assert.ok(item, `missing ${name}`);
  return new RegExp(item.match);
}

test("grammar includes Go after TGo rules", () => {
  assert.deepEqual(grammar.patterns, [
    { include: "#tgo" },
    { include: "source.go" }
  ]);
});

test("grammar matches TGo-only tokens", () => {
  const cases = [
    ["keyword.other.default.tgo", "..default"]
  ];
  for (const [name, source] of cases) {
    assert.equal(pattern(name).test(source), true, source);
  }
});

test("TGo rules do not take ordinary Go operators", () => {
  assert.equal(grammar.repository.tgo.patterns.some(
    (item) => item.name === "keyword.operator.propagation.tgo"
  ), false);
  assert.equal(grammar.repository.tgo.patterns.some(
    (item) => item.name === "storage.modifier.non-nil.tgo"
  ), false);
});

test("default marker has exactly two dots", () => {
  const marker = pattern("keyword.other.default.tgo");
  assert.equal(marker.test("..default"), true);
  assert.equal(marker.test("...default"), false);
});

test("comprehension control words keep their Go scopes", () => {
  assert.equal(grammar.repository.tgo.patterns.some(
    (item) => item.name === "keyword.control.comprehension.tgo"
  ), false);
});

test("contextual words do not use broad top-level rules", () => {
  for (const name of [
    "keyword.declaration.checked.tgo",
    "keyword.control.exhaustive.tgo"
  ]) {
    assert.equal(grammar.repository.tgo.patterns.some(
      (item) => item.name === name
    ), false);
  }
});
