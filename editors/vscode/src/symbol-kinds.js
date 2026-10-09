"use strict";

const symbolKindNames = {
  package: "Package",
  type: "Object",
  struct: "Struct",
  interface: "Interface",
  function: "Function",
  method: "Method",
  field: "Field",
  enum: "Enum",
  enumMember: "EnumMember",
  constant: "Constant",
  variable: "Variable"
};

function symbolKind(kind, values) {
  const name = symbolKindNames[kind];
  if (!name) {
    throw new Error(`unknown navigation symbol kind ${JSON.stringify(kind)}`);
  }
  return values[name];
}

module.exports = { symbolKind, symbolKindNames };
