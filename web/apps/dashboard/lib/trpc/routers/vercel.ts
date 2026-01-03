import { insertAuditLogs } from "@/lib/audit";
import { type VercelBinding, and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { Vercel } from "@unkey/vercel";
import { z } from "zod";
import { protectedProcedure, t } from "../trpc";

export const vercelRouter = t.router({
  setupProject: protectedProcedure
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
        accessToken: z.string(),
        vercelTeamId: z.string().nullable(),
        apiIds: z.object({
          development: z.string().nullable(),
          preview: z.string().nullable(),
          production: z.string().nullable(),
        }),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      // It's stupid to have to do this, we should just read `UNKEY_KEY_AUTH_ID` from the env instead
      const unkeyApi = await db.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, env().UNKEY_API_ID),
      });
      if (!unkeyApi) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "unkey api not found",
        });
      }
      if (!unkeyApi.keyAuthId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "unkey api not setup to handle keys",
        });
      }

      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "integration not found",
        });
      }
      if (!integration.workspace || integration.workspace.orgId !== ctx.tenant.id) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }

      const keyAuthId = unkeyApi.keyAuthId;
      if (!keyAuthId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "unkey api not setup to handle keys",
        });
      }

      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });
      for (const [environment, apiId] of Object.entries(input.apiIds)) {
        if (!apiId) {
          continue;
        }

        const keyId = newId("key");
        const { key, hash, start } = await newKey({
          prefix: "unkey",
          byteLength: 16,
        });
        await db.transaction(async (tx) => {
          await tx.insert(schema.keys).values({
            id: keyId,
            keyAuthId,
            name: `Vercel Integration - ${environment}`,
            hash,
            start,
            ownerId: ctx.user.id,
            workspaceId: env().UNKEY_WORKSPACE_ID,
            forWorkspaceId: integration.workspace.id,
            expires: null,
            createdAtM: Date.now(),
            remaining: null,
            deletedAtM: null,
          });
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "key.create",
            description: `Created ${keyId}`,
            resources: [
              {
                type: "key",
                id: keyId,
              },
              {
                type: "vercelIntegration",
                id: integration.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });

        const setRootKeyRes = await vercel.upsertEnvironmentVariable(
          input.projectId,
          environment,
          "UNKEY_ROOT_KEY",
          key,
          true,
        );
        if (setRootKeyRes.err) {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: setRootKeyRes.err.message,
          });
        }
        await db.transaction(async (tx) => {
          const vercelBindingId = newId("vercelBinding");
          await tx.insert(schema.vercelBindings).values({
            id: vercelBindingId,
            createdAtM: Date.now(),
            updatedAtM: Date.now(),
            resourceId: keyId,
            resourceType: "rootKey",
            vercelEnvId: setRootKeyRes.val.created.id,
            lastEditedBy: ctx.user.id,
            environment: environment as VercelBinding["environment"],
            projectId: input.projectId,
            workspaceId: integration.workspace.id,
            integrationId: integration.id,
          });
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.create",
            description: `Created ${vercelBindingId} for ${keyId}`,
            resources: [
              {
                type: "vercelBinding",
                id: vercelBindingId,
              },
              {
                type: "key",
                id: keyId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });

        // API ID stuff

        const setApiIdRes = await vercel.upsertEnvironmentVariable(
          input.projectId,
          environment,
          "UNKEY_API_ID",
          apiId,
        );
        if (setApiIdRes.err) {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: setApiIdRes.err.message,
          });
        }
        await db.transaction(async (tx) => {
          const vercelBindingId = newId("vercelBinding");
          await tx.insert(schema.vercelBindings).values({
            id: vercelBindingId,
            createdAtM: Date.now(),
            updatedAtM: null,
            deletedAtM: null,
            resourceType: "apiId",
            resourceId: apiId,
            vercelEnvId: setApiIdRes.val.created.id,
            lastEditedBy: ctx.user.id,
            environment: environment as VercelBinding["environment"],
            projectId: input.projectId,
            workspaceId: integration.workspace.id,
            integrationId: integration.id,
          });
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.create",
            description: `Created ${vercelBindingId} for ${apiId}`,
            resources: [
              {
                type: "vercelBinding",
                id: vercelBindingId,
              },
              {
                type: "api",
                id: apiId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
      }
    }),
  upsertApiId: protectedProcedure
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
        apiId: z.string(),
        environment: z.enum(["production", "preview", "development"]),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          vercelBindings: true,
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "integration not found",
        });
      }

      if (integration.workspace.orgId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }

      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const res = await vercel.upsertEnvironmentVariable(
        input.projectId,
        input.environment,
        "UNKEY_API_ID",
        input.apiId,
      );
      if (res.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: res.err.message,
        });
      }
      const existingBinding = integration.vercelBindings.find(
        (b) =>
          b.projectId === input.projectId &&
          b.environment === input.environment &&
          b.resourceType === "apiId",
      );
      if (existingBinding) {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.vercelBindings)
            .set({
              resourceId: input.apiId,
              vercelEnvId: res.val.created.id,
              updatedAtM: Date.now(),
              lastEditedBy: ctx.user.id,
            })
            .where(eq(schema.vercelBindings.id, existingBinding.id));
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.update",
            description: `Updated ${existingBinding.id}`,
            resources: [
              {
                type: "vercelBinding",
                id: existingBinding.id,
                meta: {
                  vercelEnvironment: res.val.created.id,
                },
              },
              {
                type: "api",
                id: input.apiId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
      } else {
        await db.transaction(async (tx) => {
          const vercelBindingId = newId("vercelBinding");
          await tx.insert(schema.vercelBindings).values({
            id: vercelBindingId,
            createdAtM: Date.now(),
            updatedAtM: Date.now(),
            resourceType: "apiId",
            resourceId: input.apiId,
            vercelEnvId: res.val.created.id,
            lastEditedBy: ctx.user.id,
            environment: input.environment,
            projectId: input.projectId,
            workspaceId: integration.workspace.id,
            integrationId: integration.id,
          });
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.create",
            description: `Created ${vercelBindingId} for ${input.apiId}`,
            resources: [
              {
                type: "vercelBinding",
                id: vercelBindingId,
              },
              {
                type: "api",
                id: input.apiId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
      }
    }),
  upsertNewRootKey: protectedProcedure
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
        environment: z.enum(["production", "preview", "development"]),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          vercelBindings: true,
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "integration not found",
        });
      }

      if (integration.workspace.orgId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      // It's stupid to have to do this, we should just read `UNKEY_KEY_AUTH_ID` from the env instead
      const unkeyApi = await db.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, env().UNKEY_API_ID),
      });
      if (!unkeyApi) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "unkey api not found",
        });
      }
      const keyAuthId = unkeyApi.keyAuthId;
      if (!keyAuthId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "unkey api not setup to handle keys",
        });
      }

      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const keyId = newId("key");
      const { key, hash, start } = await newKey({
        prefix: "unkey",
        byteLength: 16,
      });
      await db.transaction(async (tx) => {
        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId,
          name: `Vercel Integration - ${input.environment}`,
          hash,
          start,
          ownerId: ctx.user.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
          forWorkspaceId: integration.workspace.id,
          expires: null,
          createdAtM: Date.now(),
          remaining: null,
          deletedAtM: null,
        });
        await insertAuditLogs(tx, {
          workspaceId: integration.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "key.create",
          description: `Created ${keyId}`,
          resources: [
            {
              type: "key",
              id: keyId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });

      const res = await vercel.upsertEnvironmentVariable(
        input.projectId,
        input.environment,
        "UNKEY_ROOT_KEY",
        key,
        true,
      );
      if (res.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: res.err.message,
        });
      }
      const existingBinding = integration.vercelBindings.find(
        (b) =>
          b.projectId === input.projectId &&
          b.environment === input.environment &&
          b.resourceType === "rootKey",
      );
      if (existingBinding) {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.vercelBindings)
            .set({
              resourceId: keyId,
              vercelEnvId: res.val.created.id,
              updatedAtM: Date.now(),
              lastEditedBy: ctx.user.id,
            })
            .where(eq(schema.vercelBindings.id, existingBinding.id));
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.update",
            description: `Updated ${existingBinding.id}`,
            resources: [
              {
                type: "vercelBinding",
                id: existingBinding.id,
                meta: {
                  vercelEnvironment: res.val.created.id,
                },
              },
              {
                type: "key",
                id: keyId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
      } else {
        await db.transaction(async (tx) => {
          const vercelBindingId = newId("vercelBinding");
          await tx.insert(schema.vercelBindings).values({
            id: vercelBindingId,
            createdAtM: Date.now(),
            updatedAtM: Date.now(),
            resourceType: "rootKey",
            resourceId: keyId,
            vercelEnvId: res.val.created.id,
            lastEditedBy: ctx.user.id,
            environment: input.environment,
            projectId: input.projectId,
            workspaceId: integration.workspace.id,
            integrationId: integration.id,
          });

          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.create",
            description: `Created ${vercelBindingId} for ${keyId}`,
            resources: [
              {
                type: "vercelIntegration",
                id: integration.id,
              },
              {
                type: "vercelBinding",
                id: vercelBindingId,
                meta: {
                  environment: input.environment,
                  projectId: input.projectId,
                },
              },
              {
                type: "key",
                id: keyId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
      }
    }),
  unbind: protectedProcedure
    .input(
      z.object({
        bindingId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const binding = await db.query.vercelBindings.findFirst({
        where: eq(schema.vercelBindings.id, input.bindingId),
        with: {
          vercelIntegrations: {
            with: {
              workspace: true,
            },
          },
        },
      });
      if (!binding) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "integration not found",
        });
      }

      if (binding.vercelIntegrations.workspace.orgId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      const vercel = new Vercel({
        accessToken: binding.vercelIntegrations.accessToken,
        teamId: binding.vercelIntegrations.vercelTeamId ?? undefined,
      });

      await vercel.removeEnvironmentVariable(binding.projectId, binding.vercelEnvId);
      await db.transaction(async (tx) => {
        await tx
          .update(schema.vercelBindings)
          .set({ deletedAtM: Date.now() })
          .where(eq(schema.vercelBindings.id, binding.id));
        await insertAuditLogs(tx, {
          workspaceId: binding.vercelIntegrations.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "vercelBinding.delete",
          description: `Deleted ${binding.id}`,
          resources: [
            {
              type: "vercelBinding",
              id: binding.id,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    }),
  disconnectProject: protectedProcedure
    .input(
      z.object({
        projectId: z.string(),
        integrationId: z.string(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          vercelBindings: true,
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "integration not found",
        });
      }

      if (integration.workspace.orgId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const bindings = await db.query.vercelBindings.findMany({
        where: and(eq(schema.vercelBindings.projectId, input.projectId)),
      });

      for (const binding of bindings) {
        await vercel.removeEnvironmentVariable(binding.projectId, binding.vercelEnvId);
        await db.transaction(async (tx) => {
          await tx
            .update(schema.vercelBindings)
            .set({ deletedAtM: Date.now() })
            .where(eq(schema.vercelBindings.id, binding.id));
          await insertAuditLogs(tx, {
            workspaceId: integration.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "vercelBinding.delete",
            description: `Deleted ${binding.id}`,
            resources: [
              {
                type: "vercelBinding",
                id: binding.id,
                meta: {
                  vercelProjectId: binding.projectId,
                  vercelEnvironment: binding.environment,
                  vercelEnvironmentVariableId: binding.vercelEnvId,
                },
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        });
      }
    }),
});
