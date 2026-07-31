import { env, workosAuthEnv } from "@/lib/env";
import type { BaseAuthProvider } from "./base-provider";
import { localAuth } from "./local";
import { WorkOSAuthProvider } from "./workos";

function createAuthProvider(): BaseAuthProvider {
  switch (env().AUTH_PROVIDER) {
    case "workos":
      workosAuthEnv();
      return new WorkOSAuthProvider();
    case "local":
      return localAuth;
  }
}

export const auth = createAuthProvider();
