import { t } from "../trpc";
import { keyRouter } from "./key";
import { apiRouter } from "./api";

export const router = t.router({
  key: keyRouter,
  api: apiRouter,
});

// export type definition of API
export type Router = typeof router;
