import {
  defineCollections,
  defineConfig,
  defineDocs,
  frontmatterSchema,
} from "fumadocs-mdx/config";
import { z } from "zod";
export const { docs, meta } = defineDocs();

export const rfcs = defineCollections({
  dir: "content/rfcs",
  schema: frontmatterSchema.extend({
    authors: z.array(z.string()),
    date: z.string().date().or(z.date()),
  }),
  type: "doc",
});

export default defineConfig({
  lastModifiedTime: "git",
});
