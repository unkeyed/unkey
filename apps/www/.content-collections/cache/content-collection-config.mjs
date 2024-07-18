// content-collections.ts
import { defineCollection, defineConfig } from "@content-collections/core";
import { compileMDX } from "@content-collections/mdx";
import {
  remarkGfm,
  remarkHeading,
  remarkStructure
} from "fumadocs-core/mdx-plugins";
import GithubSlugger from "github-slugger";
var posts = defineCollection({
  name: "posts",
  directory: "content/blog",
  include: "*.mdx",
  schema: (z) => ({
    title: z.string(),
    description: z.string(),
    author: z.string(),
    date: z.string(),
    tags: z.array(z.string()),
    image: z.string().optional()
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure]
    });
    const slugger = new GithubSlugger();
    const regXHeader = /\n(?<flag>#+)\s+(?<content>.+)/g;
    const tableOfContents = Array.from(
      document.content.matchAll(regXHeader)
    ).map(({ groups }) => {
      const flag = groups?.flag;
      const content = groups?.content;
      return {
        level: flag?.length,
        text: content,
        slug: content ? slugger.slug(content) : void 0
      };
    });
    return {
      ...document,
      mdx,
      url: document._meta.path.replace("blog/", ""),
      tableOfContents
    };
  }
});
var content_collections_default = defineConfig({
  collections: [posts]
});
export {
  content_collections_default as default
};
