import { defineDocumentType, makeSource } from "contentlayer/source-files";
import rehypePrettyCode from "rehype-pretty-code";
import rehypeSlug from "rehype-slug";
import GithubSlugger from "github-slugger"
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeCodeTitles from "rehype-code-titles";
const Post = defineDocumentType(() => ({
  name: "Post",
  filePathPattern: "blog/*.mdx",
  contentType: "mdx",
  type: "Post",
  fields: {
    title: {
      type: "string",
      description: "The title of the post",
      required: true,
    },
    date: {
      type: "date",
      description: "The date of the post",
      required: true,
    },
    author: {
      type: "json",
      description: "The author of the post",
      required: true,
    },
    description: {
      type: "string",
      description: "The excerpt of the post",
      required: true,
    },
  },
  computedFields: {
    url: {
      type: "string",
      resolve: (doc) => `/${doc._raw.flattenedPath}`,
    },
    headings: {
      type: "json",
      resolve: async (doc) => {
        const slugger = new GithubSlugger()
        const regXHeader = /\n(?<flag>#{1,6})\s+(?<content>.+)/g;
        const headings = Array.from(doc.body.raw.matchAll(regXHeader)).map(
            ({ groups }) => {
              const flag = groups?.flag;
              const content = groups?.content;
              return {
                level: flag?.length == 1 ? "one"
            : flag?.length == 2 ? "two"
            : "three",
                text: content,
                slug: content ? slugger.slug(content) : undefined,
              };
            }
          );
          return headings;
      },
    },
  },
}));

const Changelog = defineDocumentType(() => ({
  name: "Changelog",
  filePathPattern: "changelog/*.mdx",
  contentType: "mdx",
  type: "Changelog",
  fields: {
    title: {
      type: "string",
      description: "The title of the changelog",
      required: true,
    },
    date: {
      type: "date",
      description: "The date of the changelog",
      required: true,
    },
    description: {
      type: "string",
      description: "The excerpt of the changelog",
      required: true,
    },
    summary: {
      type: "list",
      of: { type: "string" },
    },
    changes: {
      type: "number",
      description: "The number of changes",
      required: true,
    },
    features: {
      type: "string",
      description: "Yes or No",
      required: true,
    },
  },
  computedFields: {
    url: {
      type: "string",
      resolve: (doc) => `/changelog/${doc._raw.sourceFileName.replace(".mdx", "")}`,
    },
    date: {
      type: "string",
      resolve: (doc) => doc._raw.sourceFileName.replace(".mdx", ""),
    },
  },
}));

const Policies = defineDocumentType(() => ({
  name: "Policies",
  filePathPattern: "policies/*.mdx",
  contentType: "mdx",
  type: "Policies",
  fields: {
    title: {
      type: "string",
      description: "The title of the policies",
      required: true,
    },
  },
  computedFields: {
    url: {
      type: "string",
      resolve: (doc) => `/${doc._raw.flattenedPath}`,
    },
  },
}));

export default makeSource({
  contentDirPath: "content",
  documentTypes: [Changelog, Policies, Post],
  mdx: {
    rehypePlugins: [rehypePrettyCode,rehypeAutolinkHeadings,rehypeSlug, rehypeCodeTitles],
  },
});
