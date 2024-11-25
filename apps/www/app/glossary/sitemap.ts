import fs from "node:fs/promises";
import path from "node:path";

import { env } from "@/lib/env";
import type { MetadataRoute } from "next";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // NB: I initally tried to import allGlossaries from content-collections but it wasn't available
  const glossaryDir = path.join(process.cwd(), "content", "glossary");
  const files = await fs.readdir(glossaryDir);

  // NB: Google's limit is 50,000 URLs per sitemap, split up if necessary: https://nextjs.org/docs/app/api-reference/functions/generate-sitemaps
  return files
    .filter((file) => file.endsWith(".mdx"))
    .map((file) => ({
      url: `${env().NEXT_PUBLIC_BASE_URL}/glossary/${path.basename(file, ".mdx")}`,
      lastModified: new Date(), // TODO: Get the actual last modified date of the file -- if content-collections doesn't work from marketing-db?
    }));
}
