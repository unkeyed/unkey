import { env } from "@/lib/env";
import type { BaseAuthProvider } from "./base-provider";
import { BetterAuthProvider } from "./better-auth";
import { LocalAuthProvider } from "./local";
import { WorkOSAuthProvider } from "./workos";

type SupportedProviders = "workos" | "better-auth" | "local";

// Use globalThis to persist singleton across HMR in development
// This prevents memory leaks from recreating auth provider instances on every hot reload
const globalForAuth = globalThis as unknown as {
  authProviderInstance: BaseAuthProvider | undefined;
  authProviderInitialized: boolean;
};

// biome-ignore lint/complexity/noStaticOnlyClass: intentional; AuthProvider class is inherited/extended by other providers
class AuthProvider {
  private static get instance(): BaseAuthProvider | null {
    return globalForAuth.authProviderInstance ?? null;
  }
  private static set instance(value: BaseAuthProvider | null) {
    globalForAuth.authProviderInstance = value ?? undefined;
  }
  private static get initialized(): boolean {
    return globalForAuth.authProviderInitialized ?? false;
  }
  private static set initialized(value: boolean) {
    globalForAuth.authProviderInitialized = value;
  }

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

      case "better-auth": {
        AuthProvider.initializeBetterAuth();
        break;
      }

      case "local": {
        AuthProvider.initializeLocal();
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

  private static initializeLocal() {
    AuthProvider.instance = new LocalAuthProvider();
  }

  private static initializeBetterAuth() {
    // BetterAuthProvider validates env vars (BETTER_AUTH_SECRET, BETTER_AUTH_URL) in its constructor
    AuthProvider.instance = new BetterAuthProvider();
  }

  public static getInstance(): BaseAuthProvider {
    if (!AuthProvider.instance) {
      AuthProvider.initialize();

      if (!AuthProvider.instance) {
        throw new Error("Failed to initialize AuthProvider. Check your configuration.");
      }
    }
    return AuthProvider.instance;
  }
}

export const auth = AuthProvider.getInstance();
