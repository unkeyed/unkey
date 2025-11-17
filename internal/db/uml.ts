// dbml.ts
import * as schema from "./src/schema";
import { mysqlGenerate } from "drizzle-dbml-generator";

mysqlGenerate({ schema, out: "./schema.dbml", relational: true });
