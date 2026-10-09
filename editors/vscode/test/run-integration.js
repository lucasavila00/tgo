"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { runTests } = require("@vscode/test-electron");

async function main() {
  const extension = path.resolve(__dirname, "..");
  const repository = path.resolve(extension, "..", "..");
  const cache = path.join(extension, ".vscode-test");
  const workspace = path.join(cache, "workspace with spaces");
  const secondWorkspace = repository;
  const workspaceFile = path.join(cache, "navigation.code-workspace");
  fs.mkdirSync(cache, { recursive: true });
  fs.rmSync(workspace, { recursive: true, force: true });
  fs.cpSync(path.join(
    repository,
    "internal",
    "navigation",
    "testdata",
    "workspaces",
    "spaced workspace"
  ), workspace, { recursive: true });
  fs.writeFileSync(workspaceFile, JSON.stringify({
    folders: [{ path: workspace }, { path: secondWorkspace }]
  }));
  const helper = path.join(extension, "bin", "tgonav");
  if (!fs.existsSync(helper)) {
    throw new Error("run ./vscode.sh --package-only before extension tests");
  }
  await runTests({
    version: "1.105.1",
    extensionDevelopmentPath: extension,
    extensionTestsPath: path.join(extension, "test", "integration", "index.js"),
    extensionTestsEnv: {
      TGO_EXTENSION_PATH: extension
    },
    launchArgs: [workspaceFile, "--disable-extensions"]
  });
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
