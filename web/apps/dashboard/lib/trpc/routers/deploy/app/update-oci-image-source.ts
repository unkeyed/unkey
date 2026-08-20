import { ActorType } from "@/gen/proto/ctrl/v1/actor_pb";
import { ociImageReferenceSchema } from "@/lib/collections/deploy/apps";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { Code, ConnectError } from "@connectrpc/connect";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getCtrlClients } from "../../ctrl";

export const updateOciSource = workspaceProcedure
  .input(
    z.object({
      appId: z.string().min(1, "App is required"),
      imageReference: ociImageReferenceSchema,
    }),
  )
  .use(withRatelimit(ratelimit.update))
  .mutation(async ({ ctx, input }) => {
    try {
      await getCtrlClients().app.updateOciImageSource({
        workspaceId: ctx.workspace.id,
        appId: input.appId,
        imageReference: input.imageReference,
        actor: {
          id: ctx.user.id,
          type: ActorType.USER,
          remoteIp: ctx.audit.location,
          userAgent: ctx.audit.userAgent ?? "",
        },
      });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      if (error instanceof ConnectError) {
        if (error.code === Code.InvalidArgument) {
          throw new TRPCError({ code: "BAD_REQUEST", message: error.rawMessage });
        }
        if (error.code === Code.NotFound) {
          throw new TRPCError({ code: "NOT_FOUND", message: error.rawMessage });
        }
        if (error.code === Code.FailedPrecondition) {
          throw new TRPCError({ code: "PRECONDITION_FAILED", message: error.rawMessage });
        }
      }

      console.error("Failed to update OCI image source:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update OCI image source",
      });
    }

    return { success: true };
  });
