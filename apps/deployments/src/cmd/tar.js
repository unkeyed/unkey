const { create } = require("tar");
const fs = require("node:fs");
const path = require("node:path");

async function main() {
  const rootDir = path.resolve(path.join(__dirname, "../../../.."));
  const paths = [];
  const files = fs.readdirSync(rootDir, { encoding: "utf-8", recursive: true });
  for (const file of files) {
    const absolute = path.join(rootDir, file);
    if (!fs.statSync(absolute).isDirectory()) {
      paths.push(absolute);
    }
  }
  console.log("found", paths.length, "files");
  await create(
    {
      gzip: true,
      file: "unkey.tgz",
    },
    paths,
  );
  console.log("done");
}
main();
