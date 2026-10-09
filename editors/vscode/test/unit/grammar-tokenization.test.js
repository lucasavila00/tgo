"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const oniguruma = require("vscode-oniguruma");
const textmate = require("vscode-textmate");

const extension = path.join(__dirname, "..", "..");
const fixtures = path.join(__dirname, "..", "fixtures", "grammar");

test("TextMate scopes distinguish TGo syntax from Go syntax", async () => {
  const wasm = fs.readFileSync(require.resolve("vscode-oniguruma/release/onig.wasm"));
  await oniguruma.loadWASM(wasm.buffer.slice(
    wasm.byteOffset,
    wasm.byteOffset + wasm.byteLength
  ));
  const registry = new textmate.Registry({
    onigLib: Promise.resolve({
      createOnigScanner(patterns) {
        return new oniguruma.OnigScanner(patterns);
      },
      createOnigString(source) {
        return new oniguruma.OnigString(source);
      }
    }),
    loadGrammar(scope) {
      if (scope === "source.tgo") {
        return readGrammar(path.join(extension, "syntaxes", "tgo.tmLanguage.json"));
      }
      if (scope === "source.go") {
        return fallbackGoGrammar();
      }
      return undefined;
    }
  });
  const grammar = await registry.loadGrammar("source.tgo");
  assert.ok(grammar);

  const source = fs.readFileSync(path.join(fixtures, "source.txt"), "utf8");
  const expectations = JSON.parse(fs.readFileSync(
    path.join(fixtures, "scopes.json"), "utf8"
  ));
  const lines = tokenize(grammar, source);
  assertScopes(lines, expectations);
  const matrix = JSON.parse(fs.readFileSync(
    path.join(fixtures, "conformance.json"), "utf8"
  ));
  for (const [feature, cases] of Object.entries(matrix)) {
    assert.ok(cases.positive.length > 0, `${feature} needs a positive case`);
    assert.ok(cases.negative.length > 0, `${feature} needs a negative case`);
    assertScopes(lines, [...cases.positive, ...cases.negative]);
  }
});

function assertScopes(lines, expectations) {
  for (const expectation of expectations) {
    const lineIndex = lines.findIndex((line) => line.text.includes(expectation.line));
    assert.notEqual(lineIndex, -1, `missing fixture line ${expectation.line}`);
    const line = lines[lineIndex];
    const offset = line.text.indexOf(expectation.token);
    assert.notEqual(offset, -1, `missing token ${expectation.token}`);
    const token = line.tokens.find(
      (item) => item.startIndex <= offset && offset < item.endIndex
    );
    assert.ok(token, `missing scopes for ${expectation.token}`);
    if (expectation.has) {
      assert.ok(
        token.scopes.includes(expectation.has),
        `${expectation.token} scopes ${token.scopes.join(", ")}`
      );
    }
    if (expectation.not) {
      assert.equal(token.scopes.includes(expectation.not), false);
    }
  }
}

function readGrammar(file) {
  return textmate.parseRawGrammar(fs.readFileSync(file, "utf8"), file);
}

function tokenize(grammar, source) {
  let ruleStack = textmate.INITIAL;
  return source.split("\n").map((text) => {
    const result = grammar.tokenizeLine(text, ruleStack);
    ruleStack = result.ruleStack;
    return { text, tokens: result.tokens };
  });
}

function fallbackGoGrammar() {
  return {
    scopeName: "source.go",
    patterns: [
      { begin: "/\\*", end: "\\*/", name: "comment.block.go" },
      { match: "//.*$", name: "comment.line.go" },
      { match: "\\btype\\b", name: "keyword.type.go" },
      { match: "\\bstruct\\b", name: "keyword.struct.go" },
      { match: "\\b[A-Za-z_]\\w*(?=\\s*:(?!=))", name: "entity.name.label.go" },
      { match: "\\b(for|if|range)\\b", name: "keyword.control.go" },
      {
        match: "(?<=^\\s*func\\b[^{\\r\\n]*)(\\b[A-Za-z_]\\w*)([ \\t]+)(\\*)(?=[A-Za-z_(\\[])",
        captures: {
          1: { name: "variable.parameter.go" },
          3: { name: "keyword.operator.address.go" }
        }
      },
      { match: "\\*(?=[A-Za-z_])", name: "keyword.operator.address.go" },
      { match: "%", name: "keyword.operator.arithmetic.go" },
      { match: "\\.", name: "punctuation.other.period.go" },
      { match: "\\b[A-Z]\\w*\\b", name: "entity.name.type.go" },
      { match: "!!?", name: "keyword.operator.go" },
      { match: "\\b[A-Za-z_]\\w*\\b", name: "identifier.go" }
    ]
  };
}
