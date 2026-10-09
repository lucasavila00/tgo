"use strict";

const vscode = require("vscode");
const { NavigationClient, RequestCancelled } = require("./client");
const { byteOffsetToPosition, positionToByteOffset } = require("./positions");

const watchedPatterns = ["**/*.tgo", "**/*.go", "**/go.mod", "**/go.work"];

class ClientManager {
  constructor(output) {
    this.output = output;
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
        .get("helperPath", "tgonav");
      const command = process.env.TGO_NAV_HELPER || configured;
      client = new NavigationClient(command, folder.uri.fsPath, this.output);
      this.clients.set(key, client);
    }
    return client;
  }

  invalidate(uri) {
    const folder = vscode.workspace.getWorkspaceFolder(uri);
    if (folder) {
      const client = this.clients.get(folder.uri.toString());
      if (client) {
        client.invalidate(uri.toString());
      }
      return;
    }
    for (const client of this.clients.values()) {
      client.invalidate(uri.toString());
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
  const clients = new ClientManager(output);
  context.subscriptions.push(output, clients);
  registerProviders(context, clients);
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
  const selector = [{ language: "tgo", scheme: "file" }];
  context.subscriptions.push(
    vscode.languages.registerDefinitionProvider(selector, {
      async provideDefinition(document, position, token) {
        const client = clients.forURI(document.uri);
        const offset = positionToByteOffset(document.getText(), position);
        if (!client || offset === undefined) {
          return [];
        }
        const values = await request(client, "definition", {
          uri: document.uri.toString(), offset
        }, token);
        return convertLocations(values, token);
      }
    }),
    vscode.languages.registerReferenceProvider(selector, {
      async provideReferences(document, position, referenceContext, token) {
        const client = clients.forURI(document.uri);
        const offset = positionToByteOffset(document.getText(), position);
        if (!client || offset === undefined) {
          return [];
        }
        const values = await request(client, "references", {
          uri: document.uri.toString(),
          offset,
          includeDeclaration: referenceContext.includeDeclaration
        }, token);
        return convertLocations(values, token);
      }
    }),
    vscode.languages.registerDocumentSymbolProvider(selector, {
      async provideDocumentSymbols(document, token) {
        const client = clients.forURI(document.uri);
        if (!client) {
          return [];
        }
        const values = await request(client, "documentSymbols", {
          uri: document.uri.toString()
        }, token);
        const result = [];
        for (const value of values || []) {
          const range = await convertRange(value.range, token);
          const selection = await convertRange(value.selection, token);
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
          return client
            ? request(client, "workspaceSymbols", { query }, token)
            : [];
        }));
        const result = [];
        for (const values of batches) {
          for (const value of values || []) {
            const range = await convertRange(value.selection, token);
            if (range) {
              result.push(new vscode.SymbolInformation(
                value.name,
                symbolKind(value.kind),
                value.container || "",
                new vscode.Location(vscode.Uri.parse(value.selection.uri), range)
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

async function convertLocations(values, token) {
  const result = [];
  for (const value of values || []) {
    const range = await convertRange(value, token);
    if (range) {
      result.push(new vscode.Location(vscode.Uri.parse(value.uri), range));
    }
  }
  return result;
}

async function convertRange(value, token) {
  if (!value || token.isCancellationRequested) {
    return undefined;
  }
  const document = await vscode.workspace.openTextDocument(vscode.Uri.parse(value.uri));
  if (token.isCancellationRequested) {
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
  registerProviders,
  watchedPatterns
};
