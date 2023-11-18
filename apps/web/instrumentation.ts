import { ZodError } from "zod";
import { generateErrorMessage } from "zod-error";
import { dbEnv, env } from "./lib/env";

export function register() {
  try {
    env();
    dbEnv();
  } catch (e) {
    if (e instanceof ZodError) {
      throw new Error(generateErrorMessage(e.issues));
    }

    throw new Error("Something went wrong");
  }
}
