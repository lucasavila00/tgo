"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const { symbolKind, symbolKindNames } = require("../../src/symbol-kinds");

test("maps the complete navigation symbol kind contract", () => {
  const contract = JSON.parse(fs.readFileSync(path.join(
    __dirname, "..", "..", "..", "..", "internal", "navigation",
    "testdata", "symbol-kinds.json"
  ), "utf8"));
  const values = Object.fromEntries(contract.map(
    ({ vscode }, index) => [vscode, index]
  ));

  assert.deepEqual(Object.keys(symbolKindNames).sort(), contract.map(
    ({ wire }) => wire
  ).sort());
  for (const { wire, vscode } of contract) {
    assert.equal(symbolKind(wire, values), values[vscode]);
  }
  assert.throws(
    () => symbolKind("unknown", values),
    /unknown navigation symbol kind "unknown"/
  );
});
