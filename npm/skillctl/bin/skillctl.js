#!/usr/bin/env node
const { spawn } = require("node:child_process");
const { join } = require("node:path");
const { binaryName, packageName } = require("../lib/platform");

function resolveBinary() {
  const name = packageName(process.platform, process.arch);
  const file = binaryName(process.platform);
  try {
    return require.resolve(join(name, "bin", file));
  } catch {
    throw new Error(
      `Could not find ${name}. Install @hz/skillctl — npm should pull the matching optional dependency for ${process.platform}-${process.arch}.`,
    );
  }
}

function main() {
  let bin;
  try {
    bin = resolveBinary();
  } catch (err) {
    console.error(`skillctl: ${err.message}`);
    process.exit(1);
  }
  const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });
  child.on("error", (err) => {
    console.error(`skillctl: ${err.message}`);
    process.exit(1);
  });
  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exit(code ?? 1);
  });
}

main();
