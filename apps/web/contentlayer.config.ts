import { defineDocumentType, makeSource } from "contentlayer/source-files";
import GithubSlugger from "github-slugger";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeCodeTitles from "rehype-code-titles";
import rehypePrettyCode from "rehype-pretty-code";
import rehypeSlug from "rehype-slug";
import remarkGfm from "remark-gfm";
import { authors } from "./content/blog/authors";

const options = {
  theme: "light-plus",
  defaultLang: {
    block: "typescript",
  },
};

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
      type: "enum",
      options: Object.keys(authors),
      description: "The author id of the post",
      required: true,
    },
    description: {
      type: "string",
      description: "The excerpt of the post",
      required: true,
    },
    image: {
      type: "string",
      description: "Image of the post",
      required: false,
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
        const slugger = new GithubSlugger();
        const regXHeader = /\n(?<flag>#{1,6})\s+(?<content>.+)/g;
        const headings = Array.from(doc.body.raw.matchAll(regXHeader)).map(({ groups }) => {
          const flag = groups?.flag;
          const content = groups?.content;
          return {
            level: flag?.length === 1 ? "one" : flag?.length === 2 ? "two" : "three",
            text: content,
            slug: content ? slugger.slug(content) : undefined,
          };
        });
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
      required: false,
    },
    changes: {
      type: "number",
      description: "The number of changes",
      required: false,
    },
    features: {
      type: "string",
      description: "Yes or No",
      required: false,
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

const Job = defineDocumentType(() => ({
  name: "Job",
  filePathPattern: "jobs/*.mdx",
  contentType: "mdx",
  type: "Job",
  fields: {
    title: {
      type: "string",
      description: "The title of the job",
      required: true,
    },
    visible: {
      type: "boolean",
      description: "Whether or not the job is visible on the website",
      required: true,
    },
    description: {
      type: "string",
      description: "The excerpt of the job",
      required: true,
    },
    level: {
      type: "string",
      description: "The level of the job",
      required: false,
    },
    salary: {
      type: "string",
      description: "The salary band of the job",
      required: true,
    },
  },
  computedFields: {
    url: {
      type: "string",
      resolve: (doc) => `/careers/${doc._raw.sourceFileName.replace(".mdx", "")}`,
    },
    slug: {
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
  documentTypes: [Changelog, Policies, Post, Job],
  mdx: {
    remarkPlugins: [remarkGfm],
    rehypePlugins: [
      [rehypePrettyCode, options],
      rehypeAutolinkHeadings,
      rehypeSlug,
      rehypeCodeTitles,
    ],
  },
});
