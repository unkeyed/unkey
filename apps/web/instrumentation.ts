import { ZodError } from "zod";
import { dbEnv, env } from "./lib/env";

export function register() {
  try {
    env();
    dbEnv();
  } catch (e) {
    if (e instanceof ZodError) {
      throw new Error(e.message);
    }

    throw new Error("Something went wrong");
  }
}
