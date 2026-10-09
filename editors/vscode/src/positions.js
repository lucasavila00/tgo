"use strict";

function byteOffsetToPosition(text, byteOffset) {
  if (!Number.isInteger(byteOffset) || byteOffset < 0) {
    return undefined;
  }
  let bytes = 0;
  let line = 0;
  let character = 0;
  for (const value of text) {
    if (bytes === byteOffset) {
      return { line, character };
    }
    const width = Buffer.byteLength(value, "utf8");
    if (bytes + width > byteOffset) {
      return undefined;
    }
    bytes += width;
    if (value === "\n") {
      line++;
      character = 0;
    } else {
      character += value.length;
    }
  }
  if (bytes === byteOffset) {
    return { line, character };
  }
  return undefined;
}

function positionToByteOffset(text, position) {
  const lines = text.split("\n");
  if (position.line < 0 || position.line >= lines.length) {
    return undefined;
  }
  const line = lines[position.line];
  if (position.character < 0 || position.character > line.length) {
    return undefined;
  }
  if (position.character > 0 && position.character < line.length) {
    const previous = line.charCodeAt(position.character - 1);
    const current = line.charCodeAt(position.character);
    if (previous >= 0xd800 && previous <= 0xdbff &&
        current >= 0xdc00 && current <= 0xdfff) {
      return undefined;
    }
  }
  let offset = 0;
  for (let index = 0; index < position.line; index++) {
    offset += Buffer.byteLength(lines[index], "utf8") + 1;
  }
  offset += Buffer.byteLength(line.slice(0, position.character), "utf8");
  return offset;
}

module.exports = { byteOffsetToPosition, positionToByteOffset };
