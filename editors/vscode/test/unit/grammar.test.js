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
  const item = grammar.repository.tgo.patterns.find((value) => value.name === name);
  assert.ok(item, `missing ${name}`);
  return new RegExp(item.match);
}

test("grammar includes Go after TGo rules", () => {
  assert.deepEqual(grammar.patterns, [
    { include: "#tgo" },
    { include: "source.go" }
  ]);
});

test("grammar adds no broad TGo token rules", () => {
  const scopes = [
    "keyword.operator.propagation.tgo",
    "storage.modifier.non-nil.tgo",
    "keyword.control.comprehension.tgo",
    "keyword.declaration.checked.tgo",
    "keyword.control.exhaustive.tgo"
  ];
  for (const scope of scopes) {
    assert.equal(grammar.repository.tgo.patterns.some(
      (item) => item.name === scope
    ), false, scope);
  }
});

test("default marker has exactly two dots", () => {
  const marker = pattern("keyword.other.default.tgo");
  assert.equal(marker.test("..default"), true);
  assert.equal(marker.test("...default"), false);
});
