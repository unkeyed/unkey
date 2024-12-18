import { env } from "@/lib/env";

import type { BaseAuthProvider } from "./interface.ts";
import { LocalAuthProvider } from "./local";
import { WorkOSAuthProvider } from "./workos";

class AuthProvider {
  private static instance: BaseAuthProvider | null = null;
  private static initialized = false;

  private static initialize(): void {
    if (this.initialized) {
      return;
    }

    const authProvider = env().AUTH_PROVIDER;
    const workosApiKey = env().WORKOS_API_KEY;
    const workosClientId = env().WORKOS_CLIENT_ID;

    switch (authProvider) {
      case "workos":
        if (!workosApiKey || !workosClientId) {
          throw new Error("WORKOS_API_KEY and WORKOS_CLIENT_ID is required when using WorkOS authentication");
        }
        this.instance = new WorkOSAuthProvider({apiKey: workosApiKey, clientId: workosClientId});
        break;

      // case "local":
      //   this.instance = new LocalAuthProvider();
      //   break;

      default:
        throw new Error(`Unsupported AUTH_PROVIDER: ${authProvider}`);
    }

    this.initialized = true;
  }

  public static getInstance(): BaseAuthProvider {
    if (!this.instance) {
      this.initialize();
    }
    return this.instance!;
  }
}

export const auth = AuthProvider.getInstance();
