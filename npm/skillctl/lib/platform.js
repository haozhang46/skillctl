const packages = {
  "darwin-arm64": "@hz/skillctl-darwin-arm64",
  "darwin-x64": "@hz/skillctl-darwin-amd64",
  "linux-x64": "@hz/skillctl-linux-amd64",
  "linux-arm64": "@hz/skillctl-linux-arm64",
  "win32-x64": "@hz/skillctl-windows-amd64",
};

function packageName(platform, arch) {
  const key = `${platform}-${arch}`;
  const name = packages[key];
  if (!name) {
    throw new Error(`unsupported platform ${key}`);
  }
  return name;
}

function binaryName(platform) {
  return platform === "win32" ? "skillctl.exe" : "skillctl";
}

module.exports = { binaryName, packageName };
