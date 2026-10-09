"use strict";

const assert = require("node:assert/strict");
const { EventEmitter } = require("node:events");
const { PassThrough } = require("node:stream");
const test = require("node:test");
const { NavigationClient, RequestCancelled } = require("../../src/client");

function fakeProcess() {
  const process = new EventEmitter();
  process.stdin = new PassThrough();
  process.stdout = new PassThrough();
  process.stderr = new PassThrough();
  process.kill = () => {};
  return process;
}

test("starts the helper and resolves a response", async () => {
  const process = fakeProcess();
  let started;
  const client = new NavigationClient("tgonav", "/work space", { append() {} },
    (command, args, options) => {
      started = { command, args, options };
      return process;
    });
  const result = client.request("definition", { uri: "file:///x", offset: 1 });
  client.receive('{"id":1,"result":[]}');
  assert.deepEqual(await result, []);
  assert.equal(started.command, "tgonav");
  assert.deepEqual(started.args, ["-root", "/work space"]);
  assert.equal(started.options.cwd, "/work space");
  client.dispose();
});

test("sends cancellation and rejects the pending request", async () => {
  const process = fakeProcess();
  const token = {
    isCancellationRequested: false,
    listener: undefined,
    onCancellationRequested(listener) {
      this.listener = listener;
      return { dispose() {} };
    }
  };
  const client = new NavigationClient("tgonav", "/workspace", { append() {} },
    () => process);
  const written = [];
  process.stdin.on("data", (data) => written.push(data.toString()));
  const result = client.request("workspaceSymbols", { query: "" }, token);
  token.listener();
  await assert.rejects(result, RequestCancelled);
  assert.match(written.join(""), /"method":"cancel"/);
  client.dispose();
});

test("restarts after the helper exits", async () => {
  const processes = [fakeProcess(), fakeProcess()];
  const output = [];
  let starts = 0;
  const client = new NavigationClient("tgonav", "/workspace", {
    append(value) { output.push(value); }
  },
    () => processes[starts++]);
  const first = client.request("definition", {});
  processes[0].emit("exit", 1, null);
  await assert.rejects(first, /navigation helper exited/);
  const second = client.request("definition", {});
  processes[0].emit("error", new Error("late old-process error"));
  processes[0].stderr.write("late old-process output");
  processes[0].stdout.write("not json\n");
  processes[1].stdout.write('{"id":2,"result":[]}\n');
  assert.deepEqual(await second, []);
  assert.equal(starts, 2);
  assert.deepEqual(output, []);
  client.dispose();
});

test("handles synchronous cancellation registration", async () => {
  const process = fakeProcess();
  const token = {
    isCancellationRequested: false,
    onCancellationRequested(listener) {
      listener();
      return { dispose() {} };
    }
  };
  const client = new NavigationClient("tgonav", "/workspace", { append() {} },
    () => process);
  await assert.rejects(
    client.request("definition", {}, token),
    RequestCancelled
  );
  client.dispose();
});

test("rejects pending work after a helper input error", async () => {
  const process = fakeProcess();
  const client = new NavigationClient("tgonav", "/workspace", { append() {} },
    () => process);
  const result = client.request("definition", {});
  process.stdin.emit("error", new Error("EPIPE"));
  await assert.rejects(result, /EPIPE/);
  client.dispose();
});

test("rejects a request when helper input is closed", async () => {
  const process = fakeProcess();
  process.stdin.end();
  const client = new NavigationClient("tgonav", "/workspace", { append() {} },
    () => process);
  await assert.rejects(client.request("definition", {}), /input is closed/);
  client.dispose();
});

test("cancels pending work when the client is disposed", async () => {
  const process = fakeProcess();
  const client = new NavigationClient("tgonav", "/workspace", { append() {} },
    () => process);
  const result = client.request("definition", {});
  client.dispose();
  await assert.rejects(result, RequestCancelled);
});
