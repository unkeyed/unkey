import { addServerHandler, createResolver, defineNuxtModule } from "@nuxt/kit";
import { defu } from "defu";

export interface ModuleOptions {
  /** Whether to verify Authorization tokens automatically */
  automaticVerification: boolean;
  /**
   * Prefix for the `Authorization` header (when auto-verifying Authorization tokens).
   *
   * @default {Bearer}
   *
   * This can be overridden at runtime with `NUXT_UNKEY_AUTH_PREFIX`.
   */
  authPrefix: string;
  /**
   * Whether to auto-import a configured Unkey client which can access with `useUnkey`.
   *
   * @default {true}
   */
  registerHelper: boolean;
  /**
   * Root token for your Unkey account.
   *
   * Used to configure the Unkey client accessed with `useUnkey`.
   *
   * This can be overridden at runtime with `NUXT_UNKEY_TOKEN`.
   */
  token?: string;
}

export default defineNuxtModule<ModuleOptions>({
  meta: {
    name: "@unkey/nuxt",
    configKey: "unkey",
  },
  defaults: {
    authPrefix: "Bearer",
    automaticVerification: true,
    registerHelper: true,
  },
  setup(options, nuxt) {
    const resolver = createResolver(import.meta.url);

    nuxt.options.runtimeConfig.unkey = defu(nuxt.options.runtimeConfig.unkey as any, {
      authPrefix: options.authPrefix,
      token: options.token,
    });

    // Automatically verify all incoming requests with an Authorization header
    if (options.automaticVerification) {
      addServerHandler({
        middleware: true,
        handler: resolver.resolve("./runtime/server/middleware/unkey"),
      });
    }

    // Inject automatically-configured Unkey helper
    if (options.registerHelper) {
      nuxt.hook("nitro:config", (config) => {
        if (config.imports !== false) {
          config.imports ||= {};
          config.imports.presets ||= [];
          config.imports.presets.push({
            from: resolver.resolve("./runtime/server/utils/unkey"),
            imports: ["useUnkey"],
          });
        }
        config.typescript ||= {};
        config.typescript.tsConfig = defu(config.typescript.tsConfig, {
          include: [resolver.resolve("./runtime/server/types.d.ts")],
        });
      });

      nuxt.hook("prepare:types", ({ references }) => {
        references.push({
          path: resolver.resolve("./runtime/server/types.d.ts"),
        });
      });
    }
  },
});
