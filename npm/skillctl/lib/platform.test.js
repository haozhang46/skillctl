const { test } = require("node:test");
const assert = require("node:assert/strict");
const { binaryName, packageName } = require("./platform");

test("maps darwin arm64 to @hz/skillctl-darwin-arm64", () => {
  assert.equal(packageName("darwin", "arm64"), "@hz/skillctl-darwin-arm64");
});

test("maps darwin x64 to @hz/skillctl-darwin-amd64", () => {
  assert.equal(packageName("darwin", "x64"), "@hz/skillctl-darwin-amd64");
});

test("maps linux x64 to @hz/skillctl-linux-amd64", () => {
  assert.equal(packageName("linux", "x64"), "@hz/skillctl-linux-amd64");
});

test("maps linux arm64 to @hz/skillctl-linux-arm64", () => {
  assert.equal(packageName("linux", "arm64"), "@hz/skillctl-linux-arm64");
});

test("maps win32 x64 to @hz/skillctl-windows-amd64", () => {
  assert.equal(packageName("win32", "x64"), "@hz/skillctl-windows-amd64");
});

test("uses skillctl.exe on windows", () => {
  assert.equal(binaryName("win32"), "skillctl.exe");
});

test("uses skillctl on unix", () => {
  assert.equal(binaryName("darwin"), "skillctl");
});

test("rejects unsupported platforms", () => {
  assert.throws(() => packageName("freebsd", "x64"), /unsupported platform freebsd-x64/);
});
