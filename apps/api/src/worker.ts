import { type Env, zEnv } from "@/pkg/env";
import { newApp } from "@/pkg/hono/app";

import { registerLegacyApisListKeys } from "./routes/legacy_apis_listKeys";
import { registerV1ApisCreateApi } from "./routes/v1_apis_createApi";
import { registerV1ApisDeleteApi } from "./routes/v1_apis_deleteApi";
import { registerV1ApisGetApi } from "./routes/v1_apis_getApi";
import { registerV1ApisListKeys } from "./routes/v1_apis_listKeys";
import { registerV1KeysCreateKey } from "./routes/v1_keys_createKey";
import { registerV1KeysDeleteKey } from "./routes/v1_keys_deleteKey";
import { registerV1KeysGetKey } from "./routes/v1_keys_getKey";
import { registerV1KeysGetVerifications } from "./routes/v1_keys_getVerifications";
import { registerV1KeysUpdate } from "./routes/v1_keys_updateKey";
import { registerV1KeysUpdateRemaining } from "./routes/v1_keys_updateRemaining";
import { registerV1KeysVerifyKey } from "./routes/v1_keys_verifyKey";
import { registerV1KeysWhoAmI } from "./routes/v1_keys_whoami";
import { registerV1Liveness } from "./routes/v1_liveness";
import { registerV1RatelimitLimit } from "./routes/v1_ratelimits_limit";

// Legacy Routes
import { registerLegacyKeysCreate } from "./routes/legacy_keys_createKey";
import { registerLegacyKeysVerifyKey } from "./routes/legacy_keys_verifyKey";

// Export Durable Objects for cloudflare
export { DurableObjectUsagelimiter } from "@/pkg/usagelimit/durable_object";
export { DurableObjectRatelimiter } from "@/pkg/ratelimit/durable_object";

import { cors, init, metrics } from "@/pkg/middleware";
import type { MessageBatch } from "@cloudflare/workers-types";
import { ConsoleLogger } from "@unkey/worker-logging";
import { storeMigrationError } from "./pkg/key_migration/dlq_handler";
import { migrateKey } from "./pkg/key_migration/handler";
import type { MessageBody } from "./pkg/key_migration/message";
import { registerV1ApisDeleteKeys } from "./routes/v1_apis_deleteKeys";
import { registerV1IdentitiesCreateIdentity } from "./routes/v1_identities_createIdentity";
import { registerV1IdentitiesDeleteIdentity } from "./routes/v1_identities_deleteIdentity";
import { registerV1IdentitiesGetIdentity } from "./routes/v1_identities_getIdentity";
import { registerV1IdentitiesListIdentities } from "./routes/v1_identities_listIdentities";
import { registerV1IdentitiesUpdateIdentity } from "./routes/v1_identities_updateIdentity";
import { registerV1KeysAddPermissions } from "./routes/v1_keys_addPermissions";
import { registerV1KeysAddRoles } from "./routes/v1_keys_addRoles";
import { registerV1KeysRemovePermissions } from "./routes/v1_keys_removePermissions";
import { registerV1KeysRemoveRoles } from "./routes/v1_keys_removeRoles";
import { registerV1KeysSetPermissions } from "./routes/v1_keys_setPermissions";
import { registerV1KeysSetRoles } from "./routes/v1_keys_setRoles";
import { registerV1MigrationsCreateKeys } from "./routes/v1_migrations_createKey";
import { registerV1MigrationsEnqueueKeys } from "./routes/v1_migrations_enqueueKeys";
import { registerV1PermissionsCreatePermission } from "./routes/v1_permissions_createPermission";
import { registerV1PermissionsCreateRole } from "./routes/v1_permissions_createRole";
import { registerV1PermissionsDeletePermission } from "./routes/v1_permissions_deletePermission";
import { registerV1PermissionsDeleteRole } from "./routes/v1_permissions_deleteRole";
import { registerV1PermissionsGetPermission } from "./routes/v1_permissions_getPermission";
import { registerV1PermissionsGetRole } from "./routes/v1_permissions_getRole";
import { registerV1PermissionsListPermissions } from "./routes/v1_permissions_listPermissions";
import { registerV1PermissionsListRoles } from "./routes/v1_permissions_listRoles";
import { registerV1RatelimitDeleteOverride } from "./routes/v1_ratelimits_deleteOverride";
import { registerV1RatelimitGetOverride } from "./routes/v1_ratelimits_getOverride";
import { registerV1RatelimitListOverrides } from "./routes/v1_ratelimits_listOverrides";
import { registerV1RatelimitSetOverride } from "./routes/v1_ratelimits_setOverride";

import { registerV1AnalyticsGetVerifications } from "./routes/v1_analytics_getVerifications";

const app = newApp();

app.use("*", init());
app.use("*", cors());
app.use("*", metrics());
// app.use("*", benchmarks());

/**
 * Registering all route handlers
 */

// misc
registerV1Liveness(app);

// keys
registerV1KeysGetKey(app);
registerV1KeysWhoAmI(app);
registerV1KeysDeleteKey(app);
registerV1KeysCreateKey(app);
registerV1KeysVerifyKey(app);
registerV1KeysUpdate(app);
registerV1KeysUpdateRemaining(app);
registerV1KeysGetVerifications(app);

registerV1KeysAddPermissions(app);
registerV1KeysRemovePermissions(app);
registerV1KeysSetPermissions(app);

registerV1KeysAddRoles(app);
registerV1KeysRemoveRoles(app);
registerV1KeysSetRoles(app);

// apis
registerV1ApisGetApi(app);
registerV1ApisCreateApi(app);
registerV1ApisListKeys(app);
registerV1ApisDeleteApi(app);
registerV1ApisDeleteKeys(app);

// ratelimit
registerV1RatelimitLimit(app);
registerV1RatelimitSetOverride(app);
registerV1RatelimitListOverrides(app);
registerV1RatelimitDeleteOverride(app);
registerV1RatelimitGetOverride(app);

// migrations
registerV1MigrationsCreateKeys(app);
registerV1MigrationsEnqueueKeys(app);

// permissions
registerV1PermissionsCreatePermission(app);
registerV1PermissionsDeletePermission(app);
registerV1PermissionsGetPermission(app);
registerV1PermissionsListPermissions(app);

registerV1PermissionsCreateRole(app);
registerV1PermissionsDeleteRole(app);
registerV1PermissionsGetRole(app);
registerV1PermissionsListRoles(app);

// identities
registerV1IdentitiesCreateIdentity(app);
registerV1IdentitiesGetIdentity(app);
registerV1IdentitiesListIdentities(app);
registerV1IdentitiesUpdateIdentity(app);
registerV1IdentitiesDeleteIdentity(app);

// analytics
registerV1AnalyticsGetVerifications(app);

// legacy REST style routes
registerLegacyKeysCreate(app);
registerLegacyKeysVerifyKey(app);
registerLegacyApisListKeys(app);

const handler = {
  fetch: (req: Request, env: Env, executionCtx: ExecutionContext) => {
    const parsedEnv = zEnv.safeParse(env);
    if (!parsedEnv.success) {
      new ConsoleLogger({
        requestId: "",
        environment: env.ENVIRONMENT,
        application: "api",
      }).fatal(`BAD_ENVIRONMENT: ${parsedEnv.error.message}`);
      return Response.json(
        {
          code: "BAD_ENVIRONMENT",
          message: "Some environment variables are missing or are invalid",
          errors: parsedEnv.error,
        },
        { status: 500 },
      );
    }

    return app.fetch(req, parsedEnv.data, executionCtx);
  },

  queue: async (
    batch: MessageBatch<MessageBody>,
    env: Env,
    _executionContext: ExecutionContext,
  ) => {
    const logger = new ConsoleLogger({
      requestId: "queue",
      environment: env.ENVIRONMENT,
      application: "api",
      defaultFields: { environment: env.ENVIRONMENT },
    });

    switch (batch.queue) {
      case "key-migrations-development":
      case "key-migrations-preview":
      case "key-migrations-canary":
      case "key-migrations-production": {
        for (const message of batch.messages) {
          const result = await migrateKey(message.body, env);
          if (result.err) {
            const delaySeconds = 2 ** message.attempts;
            logger.error("Unable to migrate key", {
              message,
              error: result.err.message,
              delaySeconds,
            });
            message.retry({ delaySeconds });
          } else {
            message.ack();
          }
        }
        break;
      }
      case "key-migrations-development-dlq":
      case "key-migrations-preview-dlq":
      case "key-migrations-canary-dlq":
      case "key-migrations-production-dlq": {
        for (const message of batch.messages) {
          await storeMigrationError(message.body, env);
        }
        break;
      }
      default:
        throw new Error(`No queue handler: ${batch.queue}`);
    }
  },
} satisfies ExportedHandler<Env, MessageBody>;

// biome-ignore lint/style/noDefaultExport: Wrangler needs that
export default handler;
