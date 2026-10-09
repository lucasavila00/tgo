"use strict";

const childProcess = require("node:child_process");
const readline = require("node:readline");

class RequestCancelled extends Error {
  constructor() {
    super("request canceled");
    this.name = "RequestCancelled";
  }
}

class NavigationClient {
  constructor(command, root, output, spawn = childProcess.spawn) {
    this.command = command;
    this.root = root;
    this.output = output;
    this.spawn = spawn;
    this.process = undefined;
    this.nextID = 1;
    this.pending = new Map();
    this.closed = false;
  }

  request(method, params, token) {
    if (this.closed) {
      return Promise.reject(new Error("navigation helper is closed"));
    }
    if (token && token.isCancellationRequested) {
      return Promise.reject(new RequestCancelled());
    }
    try {
      this.start();
    } catch (error) {
      return Promise.reject(error);
    }
    const id = this.nextID++;
    return new Promise((resolve, reject) => {
      const pending = {
        resolve,
        reject,
        cancellation: { dispose() {} }
      };
      this.pending.set(id, pending);
      if (token) {
        const cancellation = token.onCancellationRequested(() => {
          if (!this.pending.delete(id)) {
            return;
          }
          pending.cancellation.dispose();
          this.write({ method: "cancel", params: { id } });
          reject(new RequestCancelled());
        });
        if (this.pending.has(id)) {
          pending.cancellation = cancellation;
        } else {
          cancellation.dispose();
        }
      }
      if (!this.pending.has(id)) {
        return;
      }
      if (!this.write({ id, method, params })) {
        this.pending.delete(id);
        pending.cancellation.dispose();
        reject(new Error("navigation helper input is closed"));
      }
    });
  }

  invalidate(uri) {
    if (!this.process || this.closed) {
      return;
    }
    this.write({ method: "invalidate", params: { uri } });
  }

  start() {
    if (this.process) {
      return;
    }
    const process = this.spawn(this.command, ["-root", this.root], {
      cwd: this.root,
      stdio: ["pipe", "pipe", "pipe"]
    });
    this.process = process;
    const lines = readline.createInterface({ input: process.stdout });
    lines.on("line", (line) => {
      if (this.process === process) {
        this.receive(line);
      }
    });
    process.stderr.on("data", (data) => {
      if (this.process === process) {
        this.output.append(data.toString());
      }
    });
    process.stdin.on("error", (error) => {
      if (this.process === process) {
        this.fail(error);
      }
    });
    process.on("error", (error) => {
      if (this.process === process) {
        this.fail(error);
      }
    });
    process.on("exit", (code, signal) => {
      if (!this.closed && this.process === process) {
        this.fail(new Error(`navigation helper exited: ${code ?? signal}`));
      }
    });
  }

  receive(line) {
    let response;
    try {
      response = JSON.parse(line);
    } catch (error) {
      this.fail(error);
      return;
    }
    const pending = this.pending.get(response.id);
    if (!pending) {
      return;
    }
    this.pending.delete(response.id);
    pending.cancellation.dispose();
    if (response.error) {
      pending.reject(new Error(response.error));
    } else {
      pending.resolve(response.result);
    }
  }

  write(message) {
    if (!this.process || !this.process.stdin.writable) {
      return false;
    }
    this.process.stdin.write(`${JSON.stringify(message)}\n`);
    return true;
  }

  fail(error) {
    const process = this.process;
    this.process = undefined;
    if (process && !process.killed) {
      process.kill();
    }
    for (const pending of this.pending.values()) {
      pending.cancellation.dispose();
      pending.reject(error);
    }
    this.pending.clear();
  }

  dispose() {
    this.closed = true;
    this.fail(new RequestCancelled());
  }
}

module.exports = { NavigationClient, RequestCancelled };
