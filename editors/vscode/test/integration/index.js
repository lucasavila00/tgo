"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vscode = require("vscode");
const oniguruma = require("vscode-oniguruma");
const textmate = require("vscode-textmate");
const { NavigationClient, RequestCancelled } = require("../../src/client");
const { documentSelector, WorkspaceClient } = require("../../src/extension");

async function run() {
  const extension = vscode.extensions.getExtension("tgo.tgo-navigation");
  assert.ok(extension, "TGo extension is absent");
  const api = await checkWorkspaceSymbolsBeforeDocumentOpen(extension);
  await checkBundledGoGrammar(extension.extensionPath);
  const folder = vscode.workspace.workspaceFolders[0];
  assert.ok(folder, "test workspace is absent");
  const uri = vscode.Uri.joinPath(folder.uri, "main.tgo");
  const document = await vscode.workspace.openTextDocument(uri);
  await vscode.window.showTextDocument(document);
  assert.equal(document.languageId, "tgo");

  const client = api.clients.forURI(uri);
  assert.ok(client, "workspace helper client is absent");
  const invalidations = [];
  const originalInvalidate = client.invalidate.bind(client);
  client.invalidate = (changed) => {
    invalidations.push(changed.toString());
    originalInvalidate(changed);
  };

  await checkRepositoryHovers();
  await checkProviders(document);
  await checkPartialWorkspace(api);
  await checkWatchers(folder, client, invalidations);
  checkRemoteURITranslation();
  await checkConfigurationRestart(api, folder, client);
  await checkCancellation();
  await checkDirtyDocument(document);
  await checkWorkspaceFolderRemoval(api);
}

async function checkWorkspaceSymbolsBeforeDocumentOpen(extension) {
  assert.equal(
    vscode.workspace.textDocuments.some((document) => document.languageId === "tgo"),
    false,
    "a TGo document is already open"
  );
  const symbols = await vscode.commands.executeCommand(
    "vscode.executeWorkspaceSymbolProvider", "Café"
  );
  assert.ok(symbols.some((symbol) => symbol.name === "Café"));
  assert.equal(extension.isActive, true);
  assert.ok(extension.exports, "TGo extension API is absent");
  return extension.exports;
}

async function checkBundledGoGrammar(extension) {
  const wasm = fs.readFileSync(require.resolve("vscode-oniguruma/release/onig.wasm"));
  await oniguruma.loadWASM(wasm.buffer.slice(
    wasm.byteOffset,
    wasm.byteOffset + wasm.byteLength
  ));
  const paths = {
    "source.tgo": path.join(extension, "syntaxes", "tgo.tmLanguage.json"),
    "source.go": path.join(
      vscode.env.appRoot,
      "extensions",
      "go",
      "syntaxes",
      "go.tmLanguage.json"
    )
  };
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
      const file = paths[scope];
      return file
        ? textmate.parseRawGrammar(fs.readFileSync(file, "utf8"), file)
        : undefined;
    }
  });
  const grammar = await registry.loadGrammar("source.tgo");
  assert.ok(grammar, "TGo TextMate grammar is absent");
  const fixtures = path.join(extension, "test", "fixtures", "grammar");
  const source = fs.readFileSync(path.join(fixtures, "source.txt"), "utf8");
  const expectations = JSON.parse(fs.readFileSync(
    path.join(fixtures, "scopes.json"), "utf8"
  ));
  const lines = tokenizeGrammar(grammar, source);
  for (const expectation of expectations) {
    assertGrammarScope(lines, expectation);
  }
  const matrix = JSON.parse(fs.readFileSync(
    path.join(fixtures, "conformance.json"), "utf8"
  ));
  for (const [feature, cases] of Object.entries(matrix)) {
    assert.ok(cases.positive.length > 0, `${feature} needs a positive case`);
    assert.ok(cases.negative.length > 0, `${feature} needs a negative case`);
    for (const expectation of [...cases.positive, ...cases.negative]) {
      assertGrammarScope(lines, expectation);
    }
  }
}

function tokenizeGrammar(grammar, source) {
  let ruleStack = textmate.INITIAL;
  return source.split("\n").map((text) => {
    const result = grammar.tokenizeLine(text, ruleStack);
    ruleStack = result.ruleStack;
    return { text, tokens: result.tokens };
  });
}

function assertGrammarScope(lines, expectation) {
  const line = lines.find((item) => item.text.includes(expectation.line));
  assert.ok(line, `missing grammar line ${expectation.line}`);
  const offset = line.text.indexOf(expectation.token);
  assert.notEqual(offset, -1, `missing grammar token ${expectation.token}`);
  const value = line.tokens.find(
    (item) => item.startIndex <= offset && offset < item.endIndex
  );
  assert.ok(value, `missing grammar scopes for ${expectation.token}`);
  const message = `${expectation.line}: ${expectation.token} scopes ${value.scopes.join(", ")}`;
  if (expectation.has) {
    assert.equal(value.scopes.includes(expectation.has), true, message);
  }
  if (expectation.not) {
    assert.equal(value.scopes.includes(expectation.not), false, message);
  }
}

async function checkRepositoryHovers() {
  const repository = path.resolve(process.env.TGO_EXTENSION_PATH, "..", "..");
  const folder = vscode.workspace.workspaceFolders.find(
    (item) => item.uri.fsPath === repository
  );
  assert.ok(folder, "repository workspace is absent");
  await checkHover(
    vscode.Uri.joinPath(folder.uri, "internal", "navigation", "navigation.tgo"),
    "Hover(",
    "func (*Engine) Hover"
  );
  await checkHover(
    vscode.Uri.joinPath(folder.uri, "internal", "navigation", "navigation.tgo"),
    "contents, ok :=",
    "var contents string"
  );
  await checkHover(
    vscode.Uri.joinPath(folder.uri, "pkg", "format", "expressions.tgo"),
    ".Tag()",
    "func (Expression) Tag() ExpressionTag"
  );
}

async function checkHover(uri, text, contents) {
  const document = await vscode.workspace.openTextDocument(uri);
  const offset = document.getText().indexOf(text);
  assert.notEqual(offset, -1, `missing hover target ${text}`);
  const hovers = await vscode.commands.executeCommand(
    "vscode.executeHoverProvider", document.uri, document.positionAt(offset + 1)
  );
  assert.equal(hovers.length, 1);
  assert.ok(hovers[0].contents.some(
    (content) => content.value.includes(contents)
  ));
}

async function checkPartialWorkspace(api) {
  const folder = vscode.workspace.workspaceFolders.find(
    (item) => path.basename(item.uri.fsPath) === "partial workspace"
  );
  assert.ok(folder, "partial workspace is absent");
  const good = vscode.Uri.joinPath(folder.uri, "good", "good.tgo");
  const goodDocument = await vscode.workspace.openTextDocument(good);
  const target = goodDocument.getText().lastIndexOf("Target");
  const position = goodDocument.positionAt(target + 1);
  const hovers = await vscode.commands.executeCommand(
    "vscode.executeHoverProvider", good, position
  );
  assert.ok(hovers[0].contents.some(
    (content) => content.value.includes("func Target() string")
  ));
  const definitions = await vscode.commands.executeCommand(
    "vscode.executeDefinitionProvider", good, position
  );
  assert.equal(definitions.length, 1);
  assert.equal(targetText(definitions[0]), "Target");
  const references = await vscode.commands.executeCommand(
    "vscode.executeReferenceProvider", good, position
  );
  assert.equal(references.length, 2);
  const documentSymbols = await vscode.commands.executeCommand(
    "vscode.executeDocumentSymbolProvider", good
  );
  assert.deepEqual(
    documentSymbols.map((symbol) => symbol.name), ["good", "Target", "Use"]
  );
  const workspaceSymbols = await vscode.commands.executeCommand(
    "vscode.executeWorkspaceSymbolProvider", "Target"
  );
  assert.ok(workspaceSymbols.some(
    (symbol) => symbol.location.uri.toString() === good.toString()
  ));

  const bad = vscode.Uri.joinPath(folder.uri, "bad", "bad.tgo");
  const badDocument = await vscode.workspace.openTextDocument(bad);
  const broken = badDocument.getText().lastIndexOf("Broken");
  const badPosition = badDocument.positionAt(broken + 1);
  assert.deepEqual(await vscode.commands.executeCommand(
    "vscode.executeDefinitionProvider", bad, badPosition
  ), []);
  assert.deepEqual(await vscode.commands.executeCommand(
    "vscode.executeHoverProvider", bad, badPosition
  ) || [], []);

  const original = await vscode.workspace.fs.readFile(bad);
  const fixed = Buffer.from(
    original.toString().replace("return Missing", "return \"value\"")
  );
  const client = api.clients.forURI(bad);
  try {
    await vscode.workspace.fs.writeFile(bad, fixed);
    client.invalidate(bad);
    const fixedDefinitions = await vscode.commands.executeCommand(
      "vscode.executeDefinitionProvider", bad, badPosition
    );
    assert.equal(fixedDefinitions.length, 1);
    assert.equal(targetText(fixedDefinitions[0]), "Broken");
  } finally {
    await vscode.workspace.fs.writeFile(bad, original);
    client.invalidate(bad);
  }
}

function checkRemoteURITranslation() {
  assert.ok(documentSelector.some(
    (selector) => selector.language === "tgo" && selector.scheme === "vscode-remote"
  ));
  const output = vscode.window.createOutputChannel("TGo URI test");
  const folder = {
    uri: vscode.Uri.parse("vscode-remote://ssh-remote+host/workspace%20with%20spaces")
  };
  const client = new WorkspaceClient(folder, "tgonav", output);
  const source = vscode.Uri.joinPath(folder.uri, "pkg", "Café.tgo");
  const helper = client.toHelperURI(source);
  assert.equal(helper, "file:///workspace%20with%20spaces/pkg/Caf%C3%A9.tgo");
  assert.equal(client.fromHelperURI(helper).toString(), source.toString());
  client.dispose();
  output.dispose();
}

async function checkConfigurationRestart(api, folder, original) {
  const configuration = vscode.workspace.getConfiguration("tgo.navigation", folder.uri);
  const helper = path.join(process.env.TGO_EXTENSION_PATH, "bin", "tgonav");
  await configuration.update(
    "helperPath", helper, vscode.ConfigurationTarget.WorkspaceFolder
  );
  let restarted;
  await waitFor(() => {
    restarted = api.clients.forURI(folder.uri);
    return restarted !== original;
  });
  await configuration.update(
    "helperPath", undefined, vscode.ConfigurationTarget.WorkspaceFolder
  );
  await waitFor(() => api.clients.forURI(folder.uri) !== restarted);
}

async function checkDirtyDocument(document) {
  for (let attempt = 1; attempt <= 3; attempt++) {
    const version = document.version;
    const edit = new vscode.WorkspaceEdit();
    edit.insert(document.uri, new vscode.Position(0, 0), `// unsaved ${attempt}\n`);
    const applied = await vscode.workspace.applyEdit(edit);
    assert.equal(
      applied,
      true,
      `dirty edit ${attempt} failed: version ${document.version}, ` +
        `started at ${version}, dirty ${document.isDirty}`
    );
    assert.equal(document.isDirty, true);
    if (attempt < 3) {
      assert.equal(await document.save(), true);
      assert.equal(document.isDirty, false);
    }
  }
  const use = document.getText().lastIndexOf("Café");
  const definitions = await vscode.commands.executeCommand(
    "vscode.executeDefinitionProvider", document.uri, document.positionAt(use + 1)
  );
  assert.deepEqual(definitions, []);
  const hovers = await vscode.commands.executeCommand(
    "vscode.executeHoverProvider", document.uri, document.positionAt(use + 1)
  );
  assert.deepEqual(hovers || [], []);
  const symbols = await vscode.commands.executeCommand(
    "vscode.executeDocumentSymbolProvider", document.uri
  );
  assert.deepEqual(symbols || [], []);
}

async function checkWorkspaceFolderRemoval(api) {
  const folders = vscode.workspace.workspaceFolders;
  assert.equal(folders.length, 4);
  const removed = folders[1];
  const client = api.clients.forURI(removed.uri);
  assert.ok(client);
  assert.equal(vscode.workspace.updateWorkspaceFolders(1, 1), true);
  await waitFor(() => !api.clients.clients.has(removed.uri.toString()));
  await assert.rejects(
    client.request("workspaceSymbols", { query: "" }),
    /navigation helper is closed/
  );
}

async function checkProviders(document) {
  const source = document.getText();
  const use = source.lastIndexOf("Café");
  const position = document.positionAt(use + 1);
  const hovers = await vscode.commands.executeCommand(
    "vscode.executeHoverProvider", document.uri, position
  );
  assert.equal(hovers.length, 1);
  assert.equal(document.getText(hovers[0].range), "Café");
  assert.ok(hovers[0].contents.some(
    (content) => content.value.includes("func Café() string")
  ));
  const definitions = await vscode.commands.executeCommand(
    "vscode.executeDefinitionProvider", document.uri, position
  );
  assert.equal(definitions.length, 1);
  const target = await vscode.workspace.openTextDocument(definitions[0].uri);
  assert.equal(target.getText(definitions[0].range), "Café");

  const references = await vscode.commands.executeCommand(
    "vscode.executeReferenceProvider", document.uri, position
  );
  assert.equal(references.length, 2);
  assert.ok(references.every(
    (reference) => reference.uri.toString() === document.uri.toString()
  ));
  assert.ok(references.every(
    (reference) => document.getText(reference.range) === "Café"
  ));
  const expectedReferenceRanges = [source.indexOf("Café"), use].map(
    (offset) => new vscode.Range(
      document.positionAt(offset), document.positionAt(offset + "Café".length)
    )
  );
  assert.deepEqual(
    references.map((reference) => rangeText(reference.range)),
    expectedReferenceRanges.map(rangeText)
  );

  const symbols = await vscode.commands.executeCommand(
    "vscode.executeDocumentSymbolProvider", document.uri
  );
  assert.ok(symbols.some((symbol) => symbol.name === "Café"));
  const workspaceSymbols = await vscode.commands.executeCommand(
    "vscode.executeWorkspaceSymbolProvider", "Café"
  );
  assert.ok(workspaceSymbols.some((symbol) => symbol.name === "Café"));
  const secondWorkspaceSymbols = await vscode.commands.executeCommand(
    "vscode.executeWorkspaceSymbolProvider", "Target"
  );
  assert.ok(secondWorkspaceSymbols.some((symbol) => symbol.name === "Target"));

  const markerUse = source.lastIndexOf("marker");
  const markerDefinitions = await vscode.commands.executeCommand(
    "vscode.executeDefinitionProvider", document.uri,
    document.positionAt(markerUse + 1)
  );
  assert.equal(markerDefinitions.length, 1);
  assert.equal(targetText(markerDefinitions[0]), "marker");
}

async function checkWatchers(folder, client, invalidations) {
  const main = vscode.Uri.joinPath(folder.uri, "main.tgo");
  const goMod = vscode.Uri.joinPath(folder.uri, "go.mod");
  const original = await vscode.workspace.fs.readFile(main);
  const originalGoMod = await vscode.workspace.fs.readFile(goMod);
  const changed = Buffer.from(original.toString().replaceAll("Before", "After"));
  try {
    await vscode.workspace.fs.writeFile(main, changed);
    await waitFor(() => invalidations.some((value) => value.endsWith("main.tgo")));
    const document = await waitForDocument(main, "After()");
    const offset = document.getText().lastIndexOf("After");
    const definitions = await vscode.commands.executeCommand(
      "vscode.executeDefinitionProvider", main, document.positionAt(offset + 1)
    );
    assert.equal(definitions.length, 1);
    assert.equal(targetText(definitions[0]), "After");

    await touchAndWait(folder, "watch.go", "package spaced\n", invalidations);
    await touchAndWait(folder, "go.work", "go 1.25\nuse .\n", invalidations);
    const before = invalidations.length;
    await vscode.workspace.fs.writeFile(
      goMod,
      Buffer.from(`${originalGoMod.toString()}\n// watcher change\n`)
    );
    await waitFor(() => invalidations.length > before);
  } finally {
    await vscode.workspace.fs.writeFile(goMod, originalGoMod);
    await vscode.workspace.fs.writeFile(main, original);
    client.invalidate(main);
    await waitForDocument(main, original.toString(), true);
  }
}

async function touchAndWait(folder, name, text, invalidations) {
  const uri = vscode.Uri.joinPath(folder.uri, name);
  const before = invalidations.length;
  await vscode.workspace.fs.writeFile(uri, Buffer.from(text));
  await waitFor(() => invalidations.length > before);
  const beforeDelete = invalidations.length;
  await vscode.workspace.fs.delete(uri);
  await waitFor(() => invalidations.length > beforeDelete);
}

async function checkCancellation() {
  const extension = process.env.TGO_EXTENSION_PATH;
  const repository = path.resolve(extension, "..", "..");
  const root = path.join(
    repository,
    "internal",
    "navigation",
    "testdata",
    "workspaces",
    "cancellation"
  );
  const output = vscode.window.createOutputChannel("TGo cancellation test");
  const helper = path.join(process.env.TGO_EXTENSION_PATH, "bin", "tgonav");
  const client = new NavigationClient(helper, root, output);
  const token = new vscode.CancellationTokenSource();
  const result = client.request("workspaceSymbols", { query: "" }, token.token);
  token.cancel();
  await assert.rejects(result, RequestCancelled);
  token.dispose();
  client.dispose();
  output.dispose();
}

function targetText(location) {
  const document = vscode.workspace.textDocuments.find(
    (item) => item.uri.toString() === location.uri.toString()
  );
  return document ? document.getText(location.range) : undefined;
}

function rangeText(range) {
  return `${range.start.line}:${range.start.character}-${range.end.line}:${range.end.character}`;
}

async function waitForDocument(uri, text, exact = false) {
  let document;
  try {
    await waitFor(async () => {
      document = await vscode.workspace.openTextDocument(uri);
      const current = document.getText();
      return exact ? current === text : current.includes(text);
    });
  } catch (error) {
    const state = document
      ? `version ${document.version}, dirty ${document.isDirty}`
      : "document not open";
    throw new Error(`timed out while waiting for ${uri}: ${state}`, {
      cause: error
    });
  }
  return document;
}

async function waitFor(condition) {
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    if (await condition()) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error("timed out while waiting for extension event");
}

module.exports = { run };
