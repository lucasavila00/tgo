"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vscode = require("vscode");
const { NavigationClient, RequestCancelled } = require("../../src/client");
const { documentSelector, WorkspaceClient } = require("../../src/extension");

async function run() {
  const extension = vscode.extensions.getExtension("tgo.tgo-navigation");
  assert.ok(extension, "TGo extension is absent");
  const api = await extension.activate();
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

  await checkProviders(document);
  await checkWatchers(folder, client, invalidations);
  checkRemoteURITranslation();
  await checkConfigurationRestart(api, folder, client);
  await checkCancellation();
  await checkDirtyDocument(document);
  await checkWorkspaceFolderRemoval(api);
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
  await configuration.update(
    "helperPath", process.env.TGO_NAV_HELPER, vscode.ConfigurationTarget.WorkspaceFolder
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
  const edit = new vscode.WorkspaceEdit();
  edit.insert(document.uri, new vscode.Position(0, 0), "// unsaved\n");
  assert.equal(await vscode.workspace.applyEdit(edit), true);
  assert.equal(document.isDirty, true);
  const use = document.getText().lastIndexOf("Café");
  const definitions = await vscode.commands.executeCommand(
    "vscode.executeDefinitionProvider", document.uri, document.positionAt(use + 1)
  );
  assert.deepEqual(definitions, []);
  const symbols = await vscode.commands.executeCommand(
    "vscode.executeDocumentSymbolProvider", document.uri
  );
  assert.deepEqual(symbols, []);
}

async function checkWorkspaceFolderRemoval(api) {
  const folders = vscode.workspace.workspaceFolders;
  assert.equal(folders.length, 2);
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
  const client = new NavigationClient(process.env.TGO_NAV_HELPER, root, output);
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

async function waitForDocument(uri, text) {
  let document;
  await waitFor(async () => {
    document = await vscode.workspace.openTextDocument(uri);
    return document.getText().includes(text);
  });
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
