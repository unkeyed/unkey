import { t } from "../trpc";
import { keyRouter } from "./key";
import { apiRouter } from "./api";
import { teamRouter } from "./team";
export const router = t.router({
  key: keyRouter,
  api: apiRouter,
  team: teamRouter,
});

// export type definition of API
export type Router = typeof router;
