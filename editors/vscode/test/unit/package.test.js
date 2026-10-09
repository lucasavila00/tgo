"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const manifest = require("../../package.json");

test("extension contributes only read-only language features", () => {
  assert.deepEqual(manifest.activationEvents, [
    "onLanguage:tgo",
    "workspaceContains:**/*.tgo"
  ]);
  assert.equal(manifest.contributes.languages[0].id, "tgo");
  assert.equal(manifest.contributes.grammars[0].scopeName, "source.tgo");
  assert.equal(manifest.contributes.commands, undefined);
  assert.deepEqual(manifest.extensionKind, ["workspace"]);
  assert.equal(manifest.browser, undefined);
  assert.equal(manifest.license, "UNLICENSED");
  assert.equal(manifest.icon, "images/icon.png");
  assert.deepEqual(manifest.contributes.languages[0].icon, {
    light: "./images/language-light.svg",
    dark: "./images/language-dark.svg"
  });
  assert.match(manifest.scripts.package, /test -x bin\/tgonav/);
  assert.match(manifest.scripts.package, /vsce package /);
  assert.equal(
    manifest.contributes.configuration.properties["tgo.navigation.helperPath"].scope,
    "resource"
  );
  assert.equal(
    manifest.contributes.configuration.properties["tgo.navigation.helperPath"].default,
    ""
  );
  assert.deepEqual(manifest.contributes.configurationDefaults["files.exclude"], {
    "**/*_tgo.go": true,
    "**/*_tgo_*.go": true
  });
});
