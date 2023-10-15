import { addEnvVarTask } from "@redwoodjs/cli-helpers";
import type { SetRootKeyOptions } from "./setRootKey";
import { Listr } from "listr2";

export const handler = async (opts: SetRootKeyOptions) => {
  const tasks = new Listr([
    addEnvVarTask(
      "UNKEY_ROOT_KEY",
      opts.rootKey,
      "Your Unkey Root Key. See: https://unkey.dev/docs/glossary#unkey-api-key-root-key"
    ),
  ]);

  tasks.run();
};
