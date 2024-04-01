import { defineDocumentType, makeSource } from "contentlayer/source-files";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeSlug from "rehype-slug";
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
      resolve: (post) => `/blog/${post._raw.flattenedPath}`,
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
    rehypePlugins: [
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
