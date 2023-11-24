import { VercelBinding, and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { Vercel } from "@unkey/vercel";
import { z } from "zod";
import { auth, t } from "../trpc";

export const vercelRouter = t.router({
  setupProject: t.procedure
    .use(auth)
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
        throw new TRPCError({ code: "NOT_FOUND", message: "unkey api not found" });
      }
      if (!unkeyApi.keyAuthId) {
        throw new TRPCError({ code: "NOT_FOUND", message: "unkey api not setup to handle keys" });
      }

      const integration = await db.query.vercelIntegrations.findFirst({
        where: eq(schema.vercelIntegrations.id, input.integrationId),
        with: {
          workspace: true,
        },
      });
      if (!integration) {
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }
      if (!integration.workspace || integration.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
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
        const { key, hash, start } = await newKey({ prefix: "unkey", byteLength: 16 });
        await db.insert(schema.keys).values({
          id: keyId,
          keyAuthId: unkeyApi.keyAuthId!,
          name: `Vercel Integration - ${environment}`,
          hash,
          start,
          ownerId: ctx.user.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
          forWorkspaceId: integration.workspace.id,
          expires: null,
          createdAt: new Date(),
          ratelimitLimit: 10,
          ratelimitRefillRate: 10,
          ratelimitRefillInterval: 1000,
          ratelimitType: "fast",
          remaining: null,
          totalUses: 0,
          deletedAt: null,
        });

        const setRootKeyRes = await vercel.upsertEnvironmentVariable(
          input.projectId,
          environment,
          "UNKEY_ROOT_KEY",
          key,
          true,
        );
        if (setRootKeyRes.error) {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: setRootKeyRes.error.message,
          });
        }
        await db.insert(schema.vercelBindings).values({
          id: newId("vercelBinding"),
          createdAt: new Date(Date.now()),
          updatedAt: new Date(Date.now()),
          resourceType: "rootKey",
          resourceId: keyId,
          vercelEnvId: setRootKeyRes.value.created.id,
          lastEditedBy: ctx.user.id,
          environment: environment as VercelBinding["environment"],
          projectId: input.projectId,
          workspaceId: integration.workspace.id,
          integrationId: integration.id,
        });

        // Api Id stuff

        const setApiIdRes = await vercel.upsertEnvironmentVariable(
          input.projectId,
          environment,
          "UNKEY_API_ID",
          apiId,
        );
        if (setApiIdRes.error) {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: setApiIdRes.error.message,
          });
        }

        await db.insert(schema.vercelBindings).values({
          id: newId("vercelBinding"),
          createdAt: new Date(Date.now()),
          updatedAt: new Date(Date.now()),
          resourceType: "apiId",
          resourceId: apiId,
          vercelEnvId: setApiIdRes.value.created.id,
          lastEditedBy: ctx.user.id,
          environment: environment as VercelBinding["environment"],
          projectId: input.projectId,
          workspaceId: integration.workspace.id,
          integrationId: integration.id,
        });
      }
    }),
  upsertApiId: t.procedure
    .use(auth)
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
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (integration.workspace.tenantId !== ctx.tenant.id) {
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
      if (res.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: res.error.message });
      }
      const existingBinding = integration.vercelBindings.find(
        (b) =>
          b.projectId === input.projectId &&
          b.environment === input.environment &&
          b.resourceType === "apiId",
      );
      if (existingBinding) {
        await db
          .update(schema.vercelBindings)
          .set({
            resourceId: input.apiId,
            vercelEnvId: res.value.created.id,
            updatedAt: new Date(),
            lastEditedBy: ctx.user.id,
          })
          .where(eq(schema.vercelBindings.id, existingBinding.id));
      } else {
        await db.insert(schema.vercelBindings).values({
          id: newId("vercelBinding"),
          createdAt: new Date(Date.now()),
          updatedAt: new Date(Date.now()),
          resourceType: "apiId",
          resourceId: input.apiId,
          vercelEnvId: res.value.created.id,
          lastEditedBy: ctx.user.id,
          environment: input.environment,
          projectId: input.projectId,
          workspaceId: integration.workspace.id,
          integrationId: integration.id,
        });
      }
    }),
  upsertNewRootKey: t.procedure
    .use(auth)
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
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (integration.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      // It's stupid to have to do this, we should just read `UNKEY_KEY_AUTH_ID` from the env instead
      const unkeyApi = await db.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, env().UNKEY_API_ID),
      });
      if (!unkeyApi) {
        throw new TRPCError({ code: "NOT_FOUND", message: "unkey api not found" });
      }
      if (!unkeyApi.keyAuthId) {
        throw new TRPCError({ code: "NOT_FOUND", message: "unkey api not setup to handle keys" });
      }

      const vercel = new Vercel({
        accessToken: integration.accessToken,
        teamId: integration.vercelTeamId ?? undefined,
      });

      const keyId = newId("key");
      const { key, hash, start } = await newKey({ prefix: "unkey", byteLength: 16 });
      await db.insert(schema.keys).values({
        id: keyId,
        keyAuthId: unkeyApi.keyAuthId!,
        name: `Vercel Integration - ${input.environment}`,
        hash,
        start,
        ownerId: ctx.user.id,
        workspaceId: env().UNKEY_WORKSPACE_ID,
        forWorkspaceId: integration.workspace.id,
        expires: null,
        createdAt: new Date(),
        ratelimitLimit: 10,
        ratelimitRefillRate: 10,
        ratelimitRefillInterval: 1000,
        ratelimitType: "fast",
        remaining: null,
        totalUses: 0,
        deletedAt: null,
      });

      const res = await vercel.upsertEnvironmentVariable(
        input.projectId,
        input.environment,
        "UNKEY_ROOT_KEY",
        key,
        true,
      );
      if (res.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: res.error.message });
      }
      const existingBinding = integration.vercelBindings.find(
        (b) =>
          b.projectId === input.projectId &&
          b.environment === input.environment &&
          b.resourceType === "rootKey",
      );
      if (existingBinding) {
        await db
          .update(schema.vercelBindings)
          .set({
            resourceId: keyId,
            vercelEnvId: res.value.created.id,
            updatedAt: new Date(),
            lastEditedBy: ctx.user.id,
          })
          .where(eq(schema.vercelBindings.id, existingBinding.id));
      } else {
        await db.insert(schema.vercelBindings).values({
          id: newId("vercelBinding"),
          createdAt: new Date(Date.now()),
          updatedAt: new Date(Date.now()),
          resourceType: "rootKey",
          resourceId: keyId,
          vercelEnvId: res.value.created.id,
          lastEditedBy: ctx.user.id,
          environment: input.environment,
          projectId: input.projectId,
          workspaceId: integration.workspace.id,
          integrationId: integration.id,
        });
      }
    }),
  unbind: t.procedure
    .use(auth)
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
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (binding.vercelIntegrations.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ code: "UNAUTHORIZED" });
      }
      const vercel = new Vercel({
        accessToken: binding.vercelIntegrations.accessToken,
        teamId: binding.vercelIntegrations.vercelTeamId ?? undefined,
      });

      await vercel.removeEnvironmentVariable(binding.projectId, binding.vercelEnvId);
      await db.delete(schema.vercelBindings).where(eq(schema.vercelBindings.id, binding.id));
    }),
  disconnectProject: t.procedure
    .use(auth)
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
        throw new TRPCError({ code: "NOT_FOUND", message: "integration not found" });
      }

      if (integration.workspace.tenantId !== ctx.tenant.id) {
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
        await db.delete(schema.vercelBindings).where(eq(schema.vercelBindings.id, binding.id));
      }
    }),
});
