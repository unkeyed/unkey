import { dbEnv, env } from "@/lib/env";
import { ZodError } from "zod";

export function register() {
  try {
    env();
    dbEnv();
  } catch (error) {
    if (error instanceof ZodError) {
      throw new Error(
        `Following environment variables have not been configured correctly \n ${error.message}`,
      );
    }
  }
}
