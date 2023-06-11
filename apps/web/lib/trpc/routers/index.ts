import { t } from "../trpc";
import { keyRouter } from "./key";
import { apiRouter } from "./api";
import { tenantRouter } from "./tenant";
export const router = t.router({
  key: keyRouter,
  api: apiRouter,
  tenant: tenantRouter,
});

// export type definition of API
export type Router = typeof router;
