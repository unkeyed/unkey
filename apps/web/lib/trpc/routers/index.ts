import { t } from "../trpc";
import { keyRouter } from "./key";
import { apiRouter } from "./api";
import { workspaceRouter } from "./workspace";
export const router = t.router({
  key: keyRouter,
  api: apiRouter,
  workspace: workspaceRouter,
});

// export type definition of API
export type Router = typeof router;
