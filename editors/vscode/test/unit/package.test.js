"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const manifest = require("../../package.json");

test("extension contributes only read-only language features", () => {
  assert.deepEqual(manifest.activationEvents, ["onLanguage:tgo"]);
  assert.equal(manifest.contributes.languages[0].id, "tgo");
  assert.equal(manifest.contributes.grammars[0].scopeName, "source.tgo");
  assert.equal(manifest.contributes.commands, undefined);
  assert.deepEqual(manifest.extensionKind, ["workspace"]);
  assert.equal(manifest.browser, undefined);
  assert.equal(
    manifest.contributes.configuration.properties["tgo.navigation.helperPath"].scope,
    "resource"
  );
});
