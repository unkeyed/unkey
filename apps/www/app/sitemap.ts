import { allChangelogs, allGlossaries, allPolicies, allPosts } from "content-collections";
import type { Changelog, Glossary, Policy, Post } from "content-collections";
import type { MetadataRoute } from "next";
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const baseUrl = "https://unkey.com";

  const posts: MetadataRoute.Sitemap = allPosts.map((post: Post) => ({
    url: `${baseUrl}/blog/${post.slug}`,
    lastModified: post.date,
  }));

  const policies: MetadataRoute.Sitemap = allPolicies.map((policy: Policy) => ({
    url: `${baseUrl}/policies/${policy.slug}`,
  }));

  const changelogs: MetadataRoute.Sitemap = allChangelogs.map((changelog: Changelog) => ({
    url: `${baseUrl}/changelog#${changelog.slug}`,
    lastModified: changelog.date,
  }));

  const glossaries: MetadataRoute.Sitemap = allGlossaries.map((glossary: Glossary) => ({
    url: `${baseUrl}/glossary/${glossary.slug}`,
    lastModified: glossary.updatedAt,
  }));

  return [
    {
      url: "https://unkey.com",
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 1,
    },
    {
      url: "https://unkey.com/about",
      lastModified: new Date(),
      changeFrequency: "yearly",
      priority: 0.6,
    },
    {
      url: "https://unkey.com/blog",
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 1,
    },
    {
      url: "https://unkey.com/changelog",
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 1,
    },
    {
      url: "https://unkey.com/glossary",
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 1,
    },
    ...posts,
    ...policies,
    ...changelogs,
    ...glossaries,
  ];
}
