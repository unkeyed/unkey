import { mysqlGenerate } from "drizzle-dbml-generator";
// dbml.ts
import * as schema from "./src/schema";

mysqlGenerate({ schema, out: "./schema.dbml", relational: true });
