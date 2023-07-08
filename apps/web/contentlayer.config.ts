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
      type: "string",
      description: "The author of the post",
      required: true,
    },
    excerpt: {
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
  documentTypes: [Changelog,Policies, Post],
});
