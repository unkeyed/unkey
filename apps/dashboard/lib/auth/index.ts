import { env } from "@/lib/env";

import type { Auth } from "./interface";
import { LocalAuth } from "./local";
import { WorkOSAuth } from "./workos";

class AuthProvider {
  private static instance: Auth<any> | null = null;
  private static initialized = false;

  private static initialize(): void {
    if (this.initialized) {
      return;
    }

    const authProvider = env().AUTH_PROVIDER;
    const workosApiKey = env().WORKOS_API_KEY;

    switch (authProvider) {
      case "workos":
        if (!workosApiKey) {
          throw new Error("WORKOS_API_KEY is required when using WorkOS authentication");
        }
        this.instance = new WorkOSAuth(workosApiKey);
        break;

      case "local":
        this.instance = new LocalAuth();
        break;

      default:
        throw new Error(`Unsupported AUTH_PROVIDER: ${authProvider}`);
    }

    this.initialized = true;
  }

  public static getInstance(): Auth<any> {
    if (!this.instance) {
      this.initialize();
    }
    return this.instance!;
  }
}

export const auth = AuthProvider.getInstance();
