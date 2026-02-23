// source.config.ts
import {
  defineCollections,
  defineConfig,
  defineDocs,
  frontmatterSchema
} from "fumadocs-mdx/config";
import { createGenerator, remarkAutoTypeTable } from "fumadocs-typescript";
var generator = createGenerator();
var { docs, meta } = defineDocs();
var components = defineCollections({
  dir: "content/design",
  schema: frontmatterSchema.extend({}),
  type: "doc"
});
var source_config_default = defineConfig({
  lastModifiedTime: "git",
  mdxOptions: {
    remarkPlugins: [[remarkAutoTypeTable, { generator }]]
  }
});
export {
  components,
  source_config_default as default,
  docs,
  meta
};
