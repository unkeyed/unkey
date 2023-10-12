import type { ExampleOptions } from "./example";

export const handler = async (opts: ExampleOptions) => {
  console.log("exampleHandler", opts);
};
