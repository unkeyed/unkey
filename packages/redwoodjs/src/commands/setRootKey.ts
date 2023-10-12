import type Yargs from "yargs";

import { handler as setRootKeyHandler } from "./setRootKeyHandler";

export interface BaseOptions {
  cwd: string | undefined;
}

export interface SetRootKeyOptions extends BaseOptions {
  rootKey: string;
  overwrite: boolean;
}

/**
 * @see: https://unkey.dev/docs/glossary#unkey-api-key-root-key
 */
export const setRootKeyCommand = {
  command: "setRootKey",
  describe:
    "Sets the Unkey Root Key in your project's .env file. See: https://unkey.dev/docs/glossary#unkey-api-key-root-key",
  builder: (yargs: Yargs.Argv<BaseOptions>) => {
    return yargs
      .option("rootKey", {
        description: "Your Unkey Root Key.",
        alias: ["root-key", "token"],
        type: "string",
        requiresArg: true,
      })
      .option("overwrite", {
        description: "Overwrite the existing root key.",
        alias: "o",
        type: "boolean",
        default: false,
      });
  },
  handler: async (opts: SetRootKeyOptions) => {
    // It would be preferable to import this dynamically, but I'm not sure how to do that.
    //const { handler: exampleHandler } = await import("./exampleHandler.js");
    await setRootKeyHandler(opts);
  },
};
