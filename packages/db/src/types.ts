import * as schema from "./schema";
import { InferModel } from "drizzle-orm";

export type Key = InferModel<typeof schema.keys>;
export type Api = InferModel<typeof schema.apis>;
export type Workspace = InferModel<typeof schema.workspaces>;
export type KeyAuth = InferModel<typeof schema.keyAuth>;
