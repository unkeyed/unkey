import { defineDocumentType, makeSource } from "contentlayer/source-files";

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
});
