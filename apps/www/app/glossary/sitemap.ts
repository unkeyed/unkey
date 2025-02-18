import { allGlossaries } from "@/.content-collections/generated";
import { env } from "@/lib/env";
import type { MetadataRoute } from "next";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // NB: Google's limit is 50,000 URLs per sitemap, split up if necessary: https://nextjs.org/docs/app/api-reference/functions/generate-sitemaps
  return allGlossaries.map((entry) => ({
    url: `${env().NEXT_PUBLIC_BASE_URL}/glossary/${entry.slug}`,
    lastModified: entry.updatedAt,
  }));
}
