import type { Logger } from "@redwoodjs/api/logger";

export * from "./keys/middleware/createApiKeyMiddleware";

export * from "./ratelimit/middleware/createRatelimitMiddleware";
export * from "./ratelimit/middleware/util";

export type * from "./keys/middleware/types";
export type * from "./ratelimit/middleware/types";

export const defaultLogger = require("abstract-logging") as Logger;
