import { requireUser, requireWorkspace, t } from "../../trpc";

export const getWorkspace = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    return ctx.workspace;
  });
