"use strict";

const path = require("node:path");
const { fileURLToPath, pathToFileURL } = require("node:url");

function helperURI(nativePath) {
  return pathToFileURL(nativePath).toString();
}

function relativeHelperPath(root, uri) {
  const value = new URL(uri);
  if (value.protocol !== "file:") {
    return undefined;
  }
  return path.relative(root, fileURLToPath(value));
}

module.exports = { helperURI, relativeHelperPath };
