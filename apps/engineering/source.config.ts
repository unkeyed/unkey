import {
  defineCollections,
  defineConfig,
  defineDocs,
  frontmatterSchema,
} from "fumadocs-mdx/config";
import { createGenerator, remarkAutoTypeTable } from "fumadocs-typescript";
import { z } from "zod";

const generator = createGenerator();

export const { docs, meta } = defineDocs();

export const rfcs = defineCollections({
  dir: "content/rfcs",
  schema: frontmatterSchema.extend({
    authors: z.array(z.string()),
    date: z.string().date().or(z.date()),
  }),
  type: "doc",
});

export const company = defineCollections({
  dir: "content/company",
  schema: frontmatterSchema.extend({}),
  type: "doc",
});

export const contributing = defineCollections({
  dir: "content/contributing",
  schema: frontmatterSchema.extend({}),
  type: "doc",
});

export const components = defineCollections({
  dir: "content/design",
  schema: frontmatterSchema.extend({}),
  type: "doc",
});

export const architecture = defineCollections({
  dir: "content/architecture",
  schema: frontmatterSchema.extend({}),
  type: "doc",
});

export const infrastructure = defineCollections({
  dir: "content/infrastructure",
  schema: frontmatterSchema.extend({}),
  type: "doc",
});

export default defineConfig({
  lastModifiedTime: "git",
  mdxOptions: {
    remarkPlugins: [[remarkAutoTypeTable, { generator }]],
  },
});
