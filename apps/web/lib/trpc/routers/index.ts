import { t } from "../trpc";
import { apiRouter } from "./api";
import { budgetRouter } from "./budgets";
import { keyRouter } from "./key";
import { keySettingsRouter } from "./keySettings";
import { plainRouter } from "./plain";
import { ratelimitRouter } from "./ratelimit";
import { rbacRouter } from "./rbac";
import { vercelRouter } from "./vercel";
import { workspaceRouter } from "./workspace";

export const router = t.router({
  key: keyRouter,
  api: apiRouter,
  workspace: workspaceRouter,
  vercel: vercelRouter,
  plain: plainRouter,
  rbac: rbacRouter,
  keySettings: keySettingsRouter,
  ratelimit: ratelimitRouter,
  budget: budgetRouter,
});

// export type definition of API
export type Router = typeof router;
