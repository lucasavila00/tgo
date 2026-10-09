"use strict";

const path = require("node:path");
const vscode = require("vscode");
const { NavigationClient, RequestCancelled } = require("./client");
const { byteOffsetToPosition, positionToByteOffset } = require("./positions");
const { helperURI, relativeHelperPath } = require("./uris");

const watchedPatterns = ["**/*.tgo", "**/*.go", "**/go.mod", "**/go.work"];
const documentSelector = [
  { language: "tgo", scheme: "file" },
  { language: "tgo", scheme: "vscode-remote" }
];

class WorkspaceClient {
  constructor(folder, command, output) {
    this.folder = folder;
    this.client = new NavigationClient(command, folder.uri.fsPath, output);
  }

  request(method, params, token) {
    return this.client.request(method, params, token);
  }

  invalidate(uri) {
    this.client.invalidate(uri ? this.toHelperURI(uri) : "");
  }

  toHelperURI(uri) {
    return helperURI(uri.fsPath);
  }

  fromHelperURI(uri) {
    const relative = relativeHelperPath(this.folder.uri.fsPath, uri);
    if (relative === undefined) {
      return vscode.Uri.parse(uri);
    }
    if (relative === "") {
      return this.folder.uri;
    }
    return vscode.Uri.joinPath(this.folder.uri, ...relative.split(path.sep));
  }

  dispose() {
    this.client.dispose();
  }
}

class ClientManager {
  constructor(output, bundledHelper) {
    this.output = output;
    this.bundledHelper = bundledHelper;
    this.clients = new Map();
  }

  forURI(uri) {
    const folder = vscode.workspace.getWorkspaceFolder(uri);
    if (!folder) {
      return undefined;
    }
    const key = folder.uri.toString();
    let client = this.clients.get(key);
    if (!client) {
      const configured = vscode.workspace
        .getConfiguration("tgo.navigation", folder.uri)
        .get("helperPath", "");
      const command = configured || this.bundledHelper;
      client = new WorkspaceClient(folder, command, this.output);
      this.clients.set(key, client);
    }
    return client;
  }

  invalidate(uri) {
    const folder = vscode.workspace.getWorkspaceFolder(uri);
    if (folder) {
      const client = this.clients.get(folder.uri.toString());
      if (client) {
        client.invalidate(uri);
      }
      return;
    }
    for (const client of this.clients.values()) {
      client.invalidate(uri);
    }
  }

  configurationChanged(event) {
    for (const [key, client] of this.clients) {
      if (event.affectsConfiguration(
        "tgo.navigation.helperPath", client.folder.uri
      )) {
        client.dispose();
        this.clients.delete(key);
      }
    }
  }

  workspaceFoldersChanged(event) {
    for (const folder of event.removed) {
      const key = folder.uri.toString();
      const client = this.clients.get(key);
      if (client) {
        client.dispose();
        this.clients.delete(key);
      }
    }
  }

  dispose() {
    for (const client of this.clients.values()) {
      client.dispose();
    }
    this.clients.clear();
  }
}

function activate(context) {
  const output = vscode.window.createOutputChannel("TGo Navigation");
  const helperName = process.platform === "win32" ? "tgonav.exe" : "tgonav";
  const bundledHelper = context.asAbsolutePath(path.join("bin", helperName));
  const clients = new ClientManager(output, bundledHelper);
  context.subscriptions.push(output, clients);
  registerProviders(context, clients);
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration(
      (event) => clients.configurationChanged(event)
    ),
    vscode.workspace.onDidChangeWorkspaceFolders(
      (event) => clients.workspaceFoldersChanged(event)
    )
  );
  for (const pattern of watchedPatterns) {
    const watcher = vscode.workspace.createFileSystemWatcher(pattern);
    watcher.onDidCreate((uri) => clients.invalidate(uri));
    watcher.onDidChange((uri) => clients.invalidate(uri));
    watcher.onDidDelete((uri) => clients.invalidate(uri));
    context.subscriptions.push(watcher);
  }
  return { clients };
}

function registerProviders(context, clients) {
  context.subscriptions.push(
    vscode.languages.registerDefinitionProvider(documentSelector, {
      async provideDefinition(document, position, token) {
        if (document.isDirty) {
          return [];
        }
        const client = clients.forURI(document.uri);
        const offset = positionToByteOffset(document.getText(), position);
        if (!client || offset === undefined) {
          return [];
        }
        const values = await request(client, "definition", {
          uri: client.toHelperURI(document.uri), offset
        }, token);
        return convertLocations(values, token, client);
      }
    }),
    vscode.languages.registerReferenceProvider(documentSelector, {
      async provideReferences(document, position, referenceContext, token) {
        if (document.isDirty) {
          return [];
        }
        const client = clients.forURI(document.uri);
        const offset = positionToByteOffset(document.getText(), position);
        if (!client || offset === undefined) {
          return [];
        }
        const values = await request(client, "references", {
          uri: client.toHelperURI(document.uri),
          offset,
          includeDeclaration: referenceContext.includeDeclaration
        }, token);
        return convertLocations(values, token, client);
      }
    }),
    vscode.languages.registerDocumentSymbolProvider(documentSelector, {
      async provideDocumentSymbols(document, token) {
        if (document.isDirty) {
          return [];
        }
        const client = clients.forURI(document.uri);
        if (!client) {
          return [];
        }
        const values = await request(client, "documentSymbols", {
          uri: client.toHelperURI(document.uri)
        }, token);
        const result = [];
        for (const value of values || []) {
          const range = await convertRange(value.range, token, client);
          const selection = await convertRange(value.selection, token, client);
          if (range && selection) {
            result.push(new vscode.DocumentSymbol(
              value.name,
              value.container || "",
              symbolKind(value.kind),
              range,
              selection
            ));
          }
        }
        return result;
      }
    }),
    vscode.languages.registerWorkspaceSymbolProvider({
      async provideWorkspaceSymbols(query, token) {
        const folders = vscode.workspace.workspaceFolders || [];
        if (folders.length === 0) {
          return [];
        }
        const batches = await Promise.all(folders.map(async (folder) => {
          const client = clients.forURI(folder.uri);
          const values = client
            ? await request(client, "workspaceSymbols", { query }, token)
            : [];
          return { client, values };
        }));
        const result = [];
        const seen = new Set();
        for (const batch of batches) {
          for (const value of batch.values || []) {
            const range = await convertRange(value.selection, token, batch.client);
            if (range) {
              const uri = batch.client.fromHelperURI(value.selection.uri);
              const key = symbolKey(value, uri, range);
              if (seen.has(key)) {
                continue;
              }
              seen.add(key);
              result.push(new vscode.SymbolInformation(
                value.name,
                symbolKind(value.kind),
                value.container || "",
                new vscode.Location(uri, range)
              ));
            }
          }
        }
        return result;
      }
    })
  );
}

async function request(client, method, params, token) {
  try {
    return await client.request(method, params, token);
  } catch (error) {
    if (error instanceof RequestCancelled) {
      throw new vscode.CancellationError();
    }
    throw error;
  }
}

async function convertLocations(values, token, client) {
  const result = [];
  for (const value of values || []) {
    const range = await convertRange(value, token, client);
    if (range) {
      result.push(new vscode.Location(client.fromHelperURI(value.uri), range));
    }
  }
  return result;
}

async function convertRange(value, token, client) {
  if (!value || token.isCancellationRequested) {
    return undefined;
  }
  const document = await vscode.workspace.openTextDocument(
    client.fromHelperURI(value.uri)
  );
  if (token.isCancellationRequested) {
    return undefined;
  }
  if (document.isDirty) {
    return undefined;
  }
  const text = document.getText();
  const start = byteOffsetToPosition(text, value.start);
  const end = byteOffsetToPosition(text, value.end);
  if (!start || !end) {
    return undefined;
  }
  return new vscode.Range(
    new vscode.Position(start.line, start.character),
    new vscode.Position(end.line, end.character)
  );
}

function symbolKey(value, uri, range) {
  return [
    value.name,
    value.kind,
    uri.toString(),
    range.start.line,
    range.start.character,
    range.end.line,
    range.end.character
  ].join("\u0000");
}

function symbolKind(kind) {
  return {
    package: vscode.SymbolKind.Package,
    type: vscode.SymbolKind.Object,
    struct: vscode.SymbolKind.Struct,
    interface: vscode.SymbolKind.Interface,
    function: vscode.SymbolKind.Function,
    method: vscode.SymbolKind.Method,
    field: vscode.SymbolKind.Field,
    enum: vscode.SymbolKind.Enum,
    enumMember: vscode.SymbolKind.EnumMember,
    constant: vscode.SymbolKind.Constant,
    variable: vscode.SymbolKind.Variable
  }[kind] || vscode.SymbolKind.Object;
}

function deactivate() {}

module.exports = {
  activate,
  deactivate,
  documentSelector,
  registerProviders,
  watchedPatterns,
  WorkspaceClient
};
