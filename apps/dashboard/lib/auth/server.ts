import { env } from "@/lib/env";
import type { BaseAuthProvider } from "./base-provider";
import { WorkOSAuthProvider } from "./workos";

type SupportedProviders = "workos" | "local";
// biome-ignore lint/complexity/noStaticOnlyClass: intentional; AuthProvider class is inherited/extended by other providers
class AuthProvider {
  private static instance: BaseAuthProvider | null = null;
  private static initialized = false;

  private static initialize(): void {
    if (AuthProvider.initialized) {
      return;
    }

    const environment = env();
    const authProvider = environment.AUTH_PROVIDER as SupportedProviders;

    switch (authProvider) {
      case "workos": {
        AuthProvider.initializeWorkOS(environment);
        break;
      }

      default:
        throw new Error(`Unsupported AUTH_PROVIDER: ${authProvider}`);
    }

    AuthProvider.initialized = true;
  }

  private static initializeWorkOS(environment: ReturnType<typeof env>) {
    const { WORKOS_API_KEY, WORKOS_CLIENT_ID } = environment;

    if (!WORKOS_API_KEY || !WORKOS_CLIENT_ID) {
      throw new Error("WORKOS_API_KEY and WORKOS_CLIENT_ID are required for WorkOS authentication");
    }

    AuthProvider.instance = new WorkOSAuthProvider({
      apiKey: WORKOS_API_KEY,
      clientId: WORKOS_CLIENT_ID,
    });
  }

  public static getInstance(): BaseAuthProvider {
    if (!AuthProvider.instance) {
      AuthProvider.initialize();
    }
    return AuthProvider.instance!;
  }
}

export const auth = AuthProvider.getInstance();
