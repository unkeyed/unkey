// content-collections.ts
import { defineCollection, defineConfig } from "@content-collections/core";
import { compileMDX } from "@content-collections/mdx";
import { remarkGfm, remarkHeading, remarkStructure } from "fumadocs-core/mdx-plugins";
import GithubSlugger from "github-slugger";

// app/glossary/data.ts
import { z } from "zod";
var categories = [
  {
    slug: "api-specification",
    title: "API Specification",
    description: "API & Web standards for defining data formats and interactions (e.g. OpenAPI, REST, HTTP Requests, etc.)"
  }
];
var categoryEnum = z.enum(
  categories.map((c) => c.slug)
);

// lib/schemas/faq-schema.ts
import { z as z2 } from "zod";
var faqSchema = z2.array(
  z2.object({
    question: z2.string(),
    answer: z2.string()
  })
);

// lib/schemas/takeaways-schema.ts
import { z as z3 } from "zod";
var takeawaysSchema = z3.object({
  tldr: z3.string(),
  definitionAndStructure: z3.array(
    z3.object({
      key: z3.string(),
      value: z3.string()
    })
  ),
  historicalContext: z3.array(
    z3.object({
      key: z3.string(),
      value: z3.string()
    })
  ),
  usageInAPIs: z3.object({
    tags: z3.array(z3.string()),
    description: z3.string()
  }),
  bestPractices: z3.array(z3.string()),
  recommendedReading: z3.array(
    z3.object({
      title: z3.string(),
      url: z3.string()
    })
  ),
  didYouKnow: z3.string()
});

// content-collections.ts
var posts = defineCollection({
  name: "posts",
  directory: "content/blog",
  include: "*.mdx",
  schema: (z4) => ({
    title: z4.string(),
    description: z4.string(),
    author: z4.string(),
    date: z4.string(),
    tags: z4.array(z4.string()),
    image: z4.string().optional()
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure]
    });
    const slugger = new GithubSlugger();
    const regXHeader = /\n(?<flag>#+)\s+(?<content>.+)/g;
    const tableOfContents = Array.from(document.content.matchAll(regXHeader)).map(({ groups }) => {
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
      slug: document._meta.path,
      url: `/blog/${document._meta.path}`,
      tableOfContents
    };
  }
});
var changelog = defineCollection({
  name: "changelog",
  directory: "content/changelog",
  include: "*.mdx",
  schema: (z4) => ({
    title: z4.string(),
    description: z4.string().optional(),
    date: z4.string(),
    tags: z4.array(z4.string()),
    image: z4.string().optional()
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure]
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path
    };
  }
});
var careers = defineCollection({
  name: "careers",
  directory: "content/careers",
  include: "*.mdx",
  schema: (z4) => ({
    title: z4.string(),
    description: z4.string(),
    visible: z4.boolean(),
    // use a range
    salary: z4.string()
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure]
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path
    };
  }
});
var policy = defineCollection({
  name: "policy",
  directory: "content/policies",
  include: "*.mdx",
  schema: (z4) => ({
    title: z4.string()
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure]
    });
    return {
      ...document,
      mdx,
      slug: document._meta.path
    };
  }
});
var glossary = defineCollection({
  name: "glossary",
  directory: "content/glossary",
  include: "*.mdx",
  schema: (z4) => ({
    title: z4.string(),
    description: z4.string(),
    h1: z4.string(),
    term: z4.string(),
    categories: z4.array(categoryEnum),
    takeaways: takeawaysSchema,
    faq: faqSchema,
    updatedAt: z4.string(),
    slug: z4.string()
  }),
  transform: async (document, context) => {
    const mdx = await compileMDX(context, document, {
      remarkPlugins: [remarkGfm, remarkHeading, remarkStructure]
    });
    const slugger = new GithubSlugger();
    const regXHeader = /(?:^|\n)(?<flag>#+)\s+(?<content>.+)/g;
    const tableOfContents = Array.from(document.content.matchAll(regXHeader)).map(({ groups }) => {
      const flag = groups?.flag;
      const content = groups?.content;
      if (flag && flag.length > 1) {
        return {
          level: flag.length,
          text: content,
          slug: content ? slugger.slug(content) : void 0
        };
      }
      return null;
    }).filter(Boolean);
    return {
      ...document,
      mdx,
      slug: document._meta.path,
      url: `/glossary/${document._meta.path}`,
      tableOfContents
    };
  }
});
var content_collections_default = defineConfig({
  collections: [posts, changelog, policy, careers, glossary]
});
export {
  content_collections_default as default
};
