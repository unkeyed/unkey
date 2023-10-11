import path from "node:path";
import fs from "node:fs";

import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts", "src/setup.ts"],
  format: ["cjs", "esm"],
  splitting: false,
  sourcemap: true,
  clean: true,
  bundle: true,
  dts: true,
  onSuccess: () => {
    console.log("Copying template files to dist...");
    const filesToCopy = ["unkey.ts.template"];
    if (!fs.existsSync(path.resolve(__dirname, "dist", "templates"))) {
      fs.mkdirSync(path.resolve(__dirname, "dist", "templates"));
    }
    for (const file of filesToCopy) {
      fs.copyFileSync(
        path.resolve(__dirname, "src", "templates", file),
        path.resolve(__dirname, "dist", "templates", file)
      );
    }
  },
});
