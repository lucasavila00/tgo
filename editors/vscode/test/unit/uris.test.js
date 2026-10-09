"use strict";

const assert = require("node:assert/strict");
const path = require("node:path");
const { fileURLToPath } = require("node:url");
const test = require("node:test");
const { helperURI, relativeHelperPath } = require("../../src/uris");

test("converts native paths to encoded helper file URIs", () => {
  const file = path.join(path.sep, "work space", "Café.tgo");
  const uri = helperURI(file);
  assert.equal(fileURLToPath(uri), file);
  assert.match(uri, /work%20space/);
  assert.match(uri, /Caf%C3%A9\.tgo/);
});

test("finds a helper file relative to its workspace", () => {
  const root = path.join(path.sep, "work space");
  const uri = helperURI(path.join(root, "pkg", "main.tgo"));
  assert.equal(relativeHelperPath(root, uri), path.join("pkg", "main.tgo"));
  assert.equal(relativeHelperPath(root, "https://example.test/main.tgo"), undefined);
});
