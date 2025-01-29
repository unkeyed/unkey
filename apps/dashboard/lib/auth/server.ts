import { env } from "@/lib/env";
import { BaseAuthProvider } from "./base-provider";
import { WorkOSAuthProvider } from "./workos";

type SupportedProviders = "workos" | "local";

class AuthProvider {
  private static instance: BaseAuthProvider | null = null;
  private static initialized = false;

  private static initialize(): void {
    if (this.initialized) return;

    const environment = env();
    const authProvider = environment.AUTH_PROVIDER as SupportedProviders;

    switch (authProvider) {
      case "workos": {
        this.initializeWorkOS(environment);
        break;
      }

      default:
        throw new Error(`Unsupported AUTH_PROVIDER: ${authProvider}`);
    }

    this.initialized = true;
  }

  private static initializeWorkOS(environment: ReturnType<typeof env>) {
    const { WORKOS_API_KEY, WORKOS_CLIENT_ID } = environment;

    if (!WORKOS_API_KEY || !WORKOS_CLIENT_ID) {
      throw new Error(
        "WORKOS_API_KEY and WORKOS_CLIENT_ID are required for WorkOS authentication"
      );
    }

    this.instance = new WorkOSAuthProvider({
      apiKey: WORKOS_API_KEY,
      clientId: WORKOS_CLIENT_ID
    });
  }

  public static getInstance(): BaseAuthProvider {
    if (!this.instance) {
      this.initialize();
    }
    return this.instance!;
  }
}

export const auth = AuthProvider.getInstance();