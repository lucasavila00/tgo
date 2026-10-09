"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const {
  byteOffsetToPosition,
  positionToByteOffset
} = require("../../src/positions");

test("converts UTF-8 bytes and UTF-16 positions", () => {
  const text = "a😀é\nCafé";
  assert.deepEqual(byteOffsetToPosition(text, 1), { line: 0, character: 1 });
  assert.deepEqual(byteOffsetToPosition(text, 5), { line: 0, character: 3 });
  assert.deepEqual(byteOffsetToPosition(text, 8), { line: 1, character: 0 });
  assert.equal(byteOffsetToPosition(text, 2), undefined);
  assert.equal(positionToByteOffset(text, { line: 0, character: 3 }), 5);
  assert.equal(positionToByteOffset(text, { line: 0, character: 2 }), undefined);
  assert.equal(positionToByteOffset(text, { line: 1, character: 4 }), 13);
});

test("rejects positions outside the document", () => {
  assert.equal(byteOffsetToPosition("x", -1), undefined);
  assert.equal(byteOffsetToPosition("x", 2), undefined);
  assert.equal(positionToByteOffset("x", { line: 1, character: 0 }), undefined);
});
