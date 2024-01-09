import { t } from "../trpc";
import { apiRouter } from "./api";
import { keyRouter } from "./key";
import { newsletterRouter } from "./newsletter";
import { plainRouter } from "./plain";
import { vercelRouter } from "./vercel";
import { workspaceRouter } from "./workspace";

export const router = t.router({
  key: keyRouter,
  api: apiRouter,
  workspace: workspaceRouter,
  vercel: vercelRouter,
  plain: plainRouter,
  newsletter: newsletterRouter,
});

// export type definition of API
export type Router = typeof router;
