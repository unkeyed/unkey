#!/usr/bin/env node
/* eslint-disable prefer-const */
import fs from "fs";
import path from "path";
import findup from "findup-sync";
import Parser from "yargs-parser";
import { hideBin } from "yargs/helpers";
import { addApiPackages, writeFile, getPaths } from "@redwoodjs/cli-helpers";
import { Listr } from "listr2";
import { execa } from "execa";
import { prettifyTemplate, updateTomlConfig } from "./utils";

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
    // Adds the Unkey sdk to the RedwoodJS api side
    addApiPackages(["@unkey/api"]),
    {
      title: "Adding unkey.ts to your api/lib directory ...",
      task: async () => {
        // Grab template
        const template = fs.readFileSync(
          path.resolve(__dirname, "templates", "unkey.ts.template"),
          "utf-8"
        );

        // Write the template to the file system and replace any existing
        const prettifiedTemplate = await prettifyTemplate(template);

        writeFile(
          path.join(getPaths().api.lib, "unkey.ts"),
          prettifiedTemplate,
          {
            existingFiles: "OVERWRITE",
          }
        );
      },
    },
    {
      title: 'Updating redwood.toml to include "@unkey/redwoodjs" as a plugin',
      task: () => {
        updateTomlConfig();
      },
    },
    {
      title: "Adding @unkey/redwoodjs to the RedwoodJS CLI",
      task: async () => {
        console.log("Adding @unkey/redwoodjs to the redwood CLI [TODO]");
        //   await execa("yarn", ["add", "@unkey/redwoodjs", "-D"], {
        //     cwd: getPaths().base,
        //     stdio: "inherit",
        //   });

        // then case use as:
        // yarn rw @unkey example blah blah
      },
    },
  ]);

  await tasks.run();
}

setup();
