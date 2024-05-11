import type { Logger } from "@redwoodjs/api/logger";

export * from "./ratelimit";

export const defaultLogger = require("abstract-logging") as Logger;
