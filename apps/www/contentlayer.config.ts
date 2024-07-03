import { defineDocumentType, makeSource } from "contentlayer/source-files";
import GithubSlugger from "github-slugger";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeCodeTitles from "rehype-code-titles";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";

export const Post = defineDocumentType(() => ({
  name: "Post",
  filePathPattern: "**/blog/*.mdx",
  contentType: "mdx",
  fields: {
    title: { type: "string", required: true },
    date: { type: "date", required: true },
    author: { type: "string", required: true },
    description: { type: "string", required: true },
    tags: {
      type: "list",
      of: { type: "string" },
    },
    image: { type: "string" },
  },
  computedFields: {
    url: {
      type: "string",
      resolve: (post) => `${post._raw.flattenedPath.replace("blog/", "")}`,
    },
    tableOfContents: {
      type: "list",
      resolve: (doc) => {
        const slugger = new GithubSlugger();
        const regXHeader = /\n(?<flag>#+)\s+(?<content>.+)/g;
        const headings = Array.from(doc.body.raw.matchAll(regXHeader)).map(({ groups }) => {
          const flag = groups?.flag;
          const content = groups?.content;
          return {
            level: flag?.length,
            text: content,
            slug: content ? slugger.slug(content) : undefined,
          };
        });
        return headings;
      },
    },
  },
}));

export const Changelog = defineDocumentType(() => ({
  name: "Changelog",
  filePathPattern: "**/changelog/*.mdx",
  contentType: "mdx",
  fields: {
    title: { type: "string", required: true },
    date: { type: "date", required: true },
    description: { type: "string", required: true },
    tags: {
      type: "list",
      of: { type: "string" },
    },
    image: { type: "string" },
  },
  computedFields: {
    tableOfContents: {
      type: "list",
      resolve: (doc) => {
        const slugger = new GithubSlugger();
        const content = doc._raw.flattenedPath.replace("changelog/", "");
        const headings = {
          text: content,
          slug: content ? slugger.slug(content) : undefined,
        };
        return headings;
      },
    },
  },
}));

export const Policy = defineDocumentType(() => ({
  name: "Policy",
  filePathPattern: "**/policies/*.mdx",
  contentType: "mdx",
  fields: {
    title: { type: "string", required: true },
  },
}));

export const Job = defineDocumentType(() => ({
  name: "Job",
  filePathPattern: "**/jobs/*.mdx",
  contentType: "mdx",
  fields: {
    title: { type: "string", required: true },
    description: { type: "string", required: true },
    visible: { type: "boolean", required: true },
    salary: { type: "string", required: true },
  },
}));

export default makeSource({
  contentDirPath: "content",
  documentTypes: [Post, Changelog, Policy, Job],
  mdx: {
    remarkPlugins: [remarkGfm],
    rehypePlugins: [
      rehypeCodeTitles,
      rehypeSlug,
      [
        rehypeAutolinkHeadings,
        {
          properties: {
            className: ["anchor"],
            "data-mdx-heading": "",
          },
        },
      ],
    ],
  },
});
