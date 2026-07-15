import { ActorType } from "@/gen/proto/ctrl/v1/actor_pb";
import { createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { db, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { Code, ConnectError } from "@connectrpc/connect";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { getCtrlClients } from "../../ctrl";

/**
 * True when the ctrl call never reached a server (service not running), as
 * opposed to ctrl reaching a decision and rejecting. Connect wraps transport
 * failures as ConnectError[unavailable]; depending on transport/version a raw
 * fetch failure can also surface unwrapped.
 */
function isCtrlConnectionError(err: unknown): boolean {
  if (err instanceof ConnectError) {
    return err.code === Code.Unavailable || err.rawMessage.includes("fetch failed");
  }
  return err instanceof Error && /fetch failed|ECONNREFUSED/i.test(err.message);
}

export const createProject = workspaceProcedure
  .input(createProjectRequestSchema)
  .use(withRatelimit(ratelimit.create))
  .mutation(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;

    // Check if slug already exists in workspace
    const existingProject = await db.query.projects.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, workspaceId), eq(table.slug, input.slug)),
      columns: {
        id: true,
      },
    });

    if (existingProject) {
      throw new TRPCError({
        code: "CONFLICT",
        message: `A project with slug "${input.slug}" already exists in this workspace`,
      });
    }

    const ctrl = getCtrlClients();

    try {
      const response = await ctrl.project.createProject({
        workspaceId,
        name: input.name,
        slug: input.slug,
        actor: {
          id: ctx.user.id,
          type: ActorType.USER,
          remoteIp: ctx.audit.location,
          userAgent: ctx.audit.userAgent ?? "",
        },
      });

      return {
        id: response.id,
      };
    } catch (err) {
      if (err instanceof TRPCError) {
        throw err;
      }

      // DEV-ONLY fallback: the dashboard-only local stack has no ctrl service,
      // so the Connect RPC dies with a transport error. Insert the row
      // directly so the create flow works end-to-end in local demos. Hard
      // gated on NODE_ENV so this can never run in a production build, and
      // only taken for connection failures — real ctrl rejections still throw.
      // Note: ctrl-side side effects (default environment, audit log) are
      // skipped, so downstream steps needing them may still require ctrl.
      if (process.env.NODE_ENV === "development" && isCtrlConnectionError(err)) {
        console.warn(
          "⚠️  [DEV ONLY] ctrl service unreachable — inserting project directly into MySQL. " +
            "This fallback never runs in production. Start ctrl for the real create path.",
        );
        const projectId = newId("project");
        await db.insert(schema.projects).values({
          id: projectId,
          workspaceId,
          name: input.name,
          slug: input.slug,
        });
        return { id: projectId };
      }

      console.error("Failed to create project:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create project. Our team has been notified of this issue.",
      });
    }
  });
