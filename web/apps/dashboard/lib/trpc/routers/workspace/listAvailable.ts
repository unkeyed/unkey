import { getAvailableWorkspaces } from "@/lib/auth/available-workspaces";
import { protectedProcedure } from "../../trpc";

export const listAvailable = protectedProcedure.query(({ ctx }) =>
  getAvailableWorkspaces(ctx.user.id),
);
