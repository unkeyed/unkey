
const fs = require("fs");
const path = require("path");


// usage
// node set-version.js <path-to-project-root> <new-version>
// e.g. node set-version.js ./packages/sdk v1.0.0

const root = process.argv[2]; // path to project root
const version = process.argv[3].replace(/^v/, ""); // new version

console.log(`Updating version=${version} in ${root}`);

const content = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf-8"));

content.version = version;

fs.writeFileSync(path.join(root, "package.json"), JSON.stringify(content, null, 2));