import { t } from "../trpc";
import { apiRouter } from "./api";
import { keyRouter } from "./key";
import { keySettingsRouter } from "./keySettings";
import { plainRouter } from "./plain";
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
});

// export type definition of API
export type Router = typeof router;
