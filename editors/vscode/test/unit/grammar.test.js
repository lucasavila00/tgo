"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const grammar = JSON.parse(fs.readFileSync(
  path.join(__dirname, "..", "..", "syntaxes", "tgo.tmLanguage.json"),
  "utf8"
));
const patterns = grammar.repository.tgo.patterns.map((item) => new RegExp(item.match));

test("grammar includes Go after TGo rules", () => {
  assert.deepEqual(grammar.patterns, [
    { include: "#tgo" },
    { include: "source.go" }
  ]);
});

test("grammar matches every TGo token", () => {
  for (const token of [
    "enum", "where", "exhaustive:", "...default", "!", "!!", "%Thing",
    "for value := range values { value }"
  ]) {
    assert.ok(patterns.some((pattern) => pattern.test(token)), token);
  }
});

test("propagation does not match Go inequality", () => {
  const item = grammar.repository.tgo.patterns.find(
    (pattern) => pattern.name === "keyword.operator.propagation.tgo"
  );
  const pattern = new RegExp(item.match);
  assert.equal(pattern.test("!"), true);
  assert.equal(pattern.test("!!"), true);
  assert.equal(pattern.test("!="), false);
});
