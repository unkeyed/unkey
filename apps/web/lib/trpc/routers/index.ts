import { t } from "../trpc";
import { apiRouter } from "./api";
import { budgetRouter } from "./budgets";
import { keyRouter } from "./key";
import { keySettingsRouter } from "./keySettings";
import { plainRouter } from "./plain";
import { rbacRouter } from "./rbac";
import { vercelRouter } from "./vercel";
import { workspaceRouter } from "./workspace";

export const router = t.router({
  api: apiRouter,
  budget: budgetRouter,
  key: keyRouter,
  keySettings: keySettingsRouter,
  plain: plainRouter,
  rbac: rbacRouter,
  vercel: vercelRouter,
  workspace: workspaceRouter,
});

// export type definition of API
export type Router = typeof router;
