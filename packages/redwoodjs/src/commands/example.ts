import type Yargs from "yargs";

import { handler as exampleHandler } from "./exampleHandler";

export interface BaseOptions {
  cwd: string | undefined;
}

export interface ExampleOptions extends BaseOptions {
  exampleOption: string | undefined;
}

export const exampleCommand = {
  command: "example",
  describe: "Example command",
  builder: (yargs: Yargs.Argv<BaseOptions>) => {
    return yargs.option("exampleOption", {
      alias: "eo",
      type: "string",
      description: "Example option",
    });
  },
  handler: async (opts: ExampleOptions) => {
    // It would be preferable to import this dynamically, but I'm not sure how to do that.
    //const { handler: exampleHandler } = await import("./exampleHandler.js");
    await exampleHandler(opts);
  },
};
