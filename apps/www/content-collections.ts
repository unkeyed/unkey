import { defineCollection, defineConfig } from "@content-collections/core";
import { compileMDX } from "@content-collections/mdx";
import { remarkGfm, remarkHeading, remarkStructure } from "fumadocs-core/mdx-plugins";
import GithubSlugger from "github-slugger";
import { categoryEnum } from "./app/glossary/data";

const posts = defineCollection({
  name: "posts",
  directory: "content/blog",
  include: "*.mdx",
  schema: (z) => ({
    title: z.string(),
    description: z.string(),
    author: z.string(),
    date: z.string(),
    tags: z.array(z.string()),
    image: z.string().optional(),
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure],
    });
    const slugger = new GithubSlugger();
    const regXHeader = /\n(?<flag>#+)\s+(?<content>.+)/g;
    const tableOfContents = Array.from(document.content.matchAll(regXHeader)).map(({ groups }) => {
      const flag = groups?.flag;
      const content = groups?.content;
      return {
        level: flag?.length,
        text: content,
        slug: content ? slugger.slug(content) : undefined,
      };
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path,
      url: `/blog/${document._meta.path}`,
      tableOfContents,
    };
  },
});

const changelog = defineCollection({
  name: "changelog",
  directory: "content/changelog",
  include: "*.mdx",
  schema: (z) => ({
    title: z.string(),
    description: z.string(),
    date: z.string(),
    tags: z.array(z.string()),
    image: z.string().optional(),
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure],
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path,
    };
  },
});

const policy = defineCollection({
  name: "policy",
  directory: "content/policies",
  include: "*.mdx",
  schema: (z) => ({
    title: z.string(),
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure],
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path,
    };
  },
});

const job = defineCollection({
  name: "job",
  directory: "content/jobs",
  include: "*.mdx",
  schema: (z) => ({
    title: z.string(),
    description: z.string(),
    visible: z.boolean(),
    salary: z.string(),
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure],
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path,
    };
  },
});

const glossary = defineCollection({
  name: "glossary",
  directory: "content/glossary",
  include: "*.mdx",
  schema: (z) => ({
    title: z.string(),
    description: z.string(),
    h1: z.string(),
    term: z.string(),
    categories: z.array(categoryEnum),
    takeaways: z.object({
      tldr: z.string(),
      definitionAndStructure: z.array(
        z.object({
          key: z.string(),
          value: z.string(),
        }),
      ),
      historicalContext: z.array(
        z.object({
          key: z.string(),
          value: z.string(),
        }),
      ),
      usageInAPIs: z.object({
        tags: z.array(z.string()),
        description: z.string(),
      }),
      bestPractices: z.array(z.string()),
      recommendedReading: z.array(
        z.object({
          title: z.string(),
          url: z.string(),
        }),
      ),
      didYouKnow: z.string(),
    }),
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure],
    });
    const slugger = new GithubSlugger();
    // This regex is different from the one in the blog post. It matches the first header without requiring a newline as well (the h1 is provided in the frontmatter)

    const regXHeader = /(?:^|\n)(?<flag>#+)\s+(?<content>.+)/g;
    const tableOfContents = Array.from(document.content.matchAll(regXHeader))
      .map(({ groups }) => {
        const flag = groups?.flag;
        const content = groups?.content;
        // Only include headers that are not the main title (h1)
        if (flag && flag.length > 1) {
          return {
            level: flag.length,
            text: content,
            slug: content ? slugger.slug(content) : undefined,
          };
        }
        return null;
      })
      .filter(Boolean); // Remove null entries
    return {
      ...document,
      mdx,
      slug: document._meta.path,
      url: `/glossary/${document._meta.path}`,
      tableOfContents,
    };
  },
});

export default defineConfig({
  collections: [posts, changelog, policy, job, glossary],
});
