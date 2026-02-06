import {
  defineCollections,
  defineConfig,
  defineDocs,
  frontmatterSchema,
} from "fumadocs-mdx/config";

import { createGenerator, remarkAutoTypeTable } from "fumadocs-typescript";

const generator = createGenerator();

export const { docs, meta } = defineDocs();

export const components = defineCollections({
  dir: "content/design",
  schema: frontmatterSchema.extend({}),
  type: "doc",
});

export const { docs: packageDocs, meta: packageMeta } = defineDocs({
  dir: "content/packages",
});

export default defineConfig({
  lastModifiedTime: "git",
  mdxOptions: {
    remarkPlugins: [[remarkAutoTypeTable, { generator }]],
  },
});
