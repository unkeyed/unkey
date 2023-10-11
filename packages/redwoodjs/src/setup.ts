#!/usr/bin/env node
/* eslint-disable prefer-const */
import fs from "fs";
import path from "path";
import findup from "findup-sync";
import Parser from "yargs-parser";
import { hideBin } from "yargs/helpers";
import { addApiPackages, writeFile, getPaths } from "@redwoodjs/cli-helpers";
import { Listr } from "listr2";

// We take in yargs because we want to allow `--cwd` to be passed in, similar to the redwood cli itself.
let { cwd, help } = Parser(hideBin(process.argv));

// Redwood must set the `RWJS_CWD` env var to the project's root directory so that the internal libraries
// know where to look for files.
cwd ??= process.env["RWJS_CWD"];
try {
  if (cwd) {
    // `cwd` was set by the `--cwd` option or the `RWJS_CWD` env var. In this case,
    // we don't want to find up for a `redwood.toml` file. The `redwood.toml` should just be in that directory.
    if (!fs.existsSync(path.join(cwd, "redwood.toml")) && !help) {
      throw new Error(`Couldn't find a "redwood.toml" file in ${cwd}`);
    }
  } else {
    // `cwd` wasn't set. Odds are they're in a Redwood project,
    // but they could be in ./api or ./web, so we have to find up to be sure.

    const redwoodTOMLPath = findup("redwood.toml", { cwd: process.cwd() });

    if (!redwoodTOMLPath && !help) {
      throw new Error(
        `Couldn't find up a "redwood.toml" file from ${process.cwd()}`
      );
    }

    if (redwoodTOMLPath) {
      cwd = path.dirname(redwoodTOMLPath);
    }
  }
} catch (error) {
  if (error instanceof Error) {
    console.error(error.message);
  }

  process.exit(1);
}
process.env["RWJS_CWD"] = cwd;

// Run run the setup function
async function setup() {
  const tasks = new Listr([
    // add the unkey sdk
    addApiPackages(["@unkey/api"]),
    {
      title: "Adding unkey.ts",
      task: async () => {
        const template = fs.readFileSync(
          path.resolve(__dirname, "templates", "unkey.ts.template"),
          "utf-8"
        );

        writeFile(path.join(getPaths().api.lib, "unkey.ts"), template, {
          existingFiles: "OVERWRITE",
        });

        // Format the file with prettier?
      },
    },
  ]);

  await tasks.run();
}
setup();

// copy over lb/unkey.ts template
