"use strict";

const fs = require("node:fs");
const path = require("node:path");
const childProcess = require("node:child_process");
const { runTests } = require("@vscode/test-electron");

async function main() {
  const extension = path.resolve(__dirname, "..");
  const repository = path.resolve(extension, "..", "..");
  const cache = path.join(extension, ".vscode-test");
  const helper = path.join(cache, "tgonav");
  const workspace = path.join(cache, "workspace with spaces");
  const secondWorkspace = path.join(cache, "second workspace");
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
  fs.rmSync(secondWorkspace, { recursive: true, force: true });
  fs.cpSync(path.join(
    repository,
    "internal",
    "navigation",
    "testdata",
    "workspaces",
    "basic"
  ), secondWorkspace, { recursive: true });
  fs.writeFileSync(workspaceFile, JSON.stringify({
    folders: [{ path: workspace }, { path: secondWorkspace }]
  }));
  childProcess.execFileSync("go", ["build", "-o", helper, "./cmd/tgonav"], {
    cwd: repository,
    stdio: "inherit"
  });
  await runTests({
    version: "1.105.1",
    extensionDevelopmentPath: extension,
    extensionTestsPath: path.join(extension, "test", "integration", "index.js"),
    extensionTestsEnv: {
      TGO_EXTENSION_PATH: extension,
      TGO_NAV_HELPER: helper
    },
    launchArgs: [workspaceFile, "--disable-extensions"]
  });
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
