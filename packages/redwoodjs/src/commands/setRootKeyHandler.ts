import type { SetRootKeyOptions } from "./setRootKey";
import { addEnvironmentVariablesToEnvFile } from "../utils";
import type { EnvVar } from "../utils";

export const handler = async (opts: SetRootKeyOptions) => {
  console.log("exampleHandler", opts);

  const newEnvVariables: EnvVar[] = [
    { key: "UNKEY_ROOT_KEY", value: opts.rootKey, overwrite: opts.overwrite },
  ];

  addEnvironmentVariablesToEnvFile(newEnvVariables);
};
