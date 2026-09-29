import { db } from "@/lib/db";
import { privateNetworking } from "@/lib/flags";
import { TRPCError } from "@trpc/server";
import type { z } from "zod";
import { requireApp } from "./access";
import type { target } from "./schemas";
import { bindingHostVariable, defaultBindingName } from "./validation";
import type { bindingEndpointsSchema } from "./validation";

type Endpoints = z.infer<typeof bindingEndpointsSchema>;
type Target = z.infer<typeof target>;

export async function validateEndpoints(workspaceId: string, projectId: string, input: Endpoints) {
  const [, targetApp, callerEnvironment, existing] = await Promise.all([
    requireApp(workspaceId, projectId, input.appId),
    requireApp(workspaceId, projectId, input.targetAppId),
    db.query.environments.findFirst({
      columns: { id: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.id, input.environmentId),
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, input.appId),
        ),
    }),
    db.query.appBindings.findFirst({
      columns: { id: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, input.appId),
          eq(t.environmentId, input.environmentId),
          eq(t.resourceType, "app"),
          eq(t.resourceId, input.targetAppId),
        ),
    }),
  ]);
  if (!callerEnvironment) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Caller environment not found.",
    });
  }
  if (existing) {
    throw new TRPCError({
      code: "CONFLICT",
      message: "This app is already connected.",
    });
  }
  return { targetSlug: targetApp.slug };
}

export async function validateTarget(
  workspaceId: string,
  projectId: string,
  targetAppId: string,
  input: Target,
) {
  if (input.targetType === "environment") {
    const env = await db.query.environments.findFirst({
      columns: { id: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.id, input.targetEnvironmentId),
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, targetAppId),
        ),
    });
    if (!env) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Select an environment from the target app.",
      });
    }
  }
  if (input.targetType === "deployment") {
    const deployment = await db.query.deployments.findFirst({
      columns: { id: true, status: true, firstReadyAt: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.id, input.targetDeploymentId),
          eq(t.workspaceId, workspaceId),
          eq(t.projectId, projectId),
          eq(t.appId, targetAppId),
        ),
    });
    if (
      !deployment ||
      !deployment.firstReadyAt ||
      !["ready", "stopped"].includes(deployment.status)
    ) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Select an approved deployment that reached ready.",
      });
    }
  }
}

export async function requirePrivateNetworking() {
  if (!(await privateNetworking())) {
    throw new TRPCError({
      code: "FORBIDDEN",
      message: "Private networking isn't enabled for this workspace.",
    });
  }
}

async function takenNames(
  workspaceId: string,
  scope: { appId: string; environmentId: string },
  excludeId?: string,
) {
  const [bindings, callerApp] = await Promise.all([
    db.query.appBindings.findMany({
      columns: { id: true, name: true },
      where: (t, { and, eq }) =>
        and(
          eq(t.workspaceId, workspaceId),
          eq(t.appId, scope.appId),
          eq(t.environmentId, scope.environmentId),
        ),
    }),
    db.query.apps.findFirst({
      columns: { slug: true },
      where: (t, { and, eq }) => and(eq(t.id, scope.appId), eq(t.workspaceId, workspaceId)),
    }),
  ]);
  const names = new Set(bindings.filter((b) => b.id !== excludeId).map((b) => b.name));
  return (name: string) => name === callerApp?.slug || names.has(name);
}

export async function pickDefaultName(workspaceId: string, scope: Endpoints, targetSlug: string) {
  const name = defaultBindingName(targetSlug, await takenNames(workspaceId, scope));
  if (!name) {
    throw new TRPCError({
      code: "CONFLICT",
      message: "Choose a name for this binding.",
    });
  }
  return name;
}

export async function requireUnusedName(
  workspaceId: string,
  scope: { appId: string; environmentId: string },
  name: string,
  excludeId?: string,
) {
  const isTaken = await takenNames(workspaceId, scope, excludeId);
  if (isTaken(name)) {
    throw new TRPCError({
      code: "CONFLICT",
      message: `${name} or ${bindingHostVariable(name)} is already used in this environment.`,
    });
  }
  return name;
}
