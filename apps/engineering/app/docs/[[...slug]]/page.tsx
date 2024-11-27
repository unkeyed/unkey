import { source } from "@/app/source";
import { getGithubLastEdit } from "fumadocs-core/server";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/page";
import type { Metadata } from "next";

import { notFound } from "next/navigation";

export default async function Page({
  params,
}: {
  params: { slug?: string[] };
}) {
  const page = source.getPage(params.slug);
  if (!page) {
    notFound();
  }

  const lastUpdate = await getGithubLastEdit({
    owner: "unkeyed",
    repo: "unkey",
    path: `apps/engineering/content/docs/${page.file.path}`,
  });

  const MDX = page.data.body;

  return (
    <DocsPage
      toc={page.data.toc}
      tableOfContent={{
        style: "clerk",
        single: true,
      }}
      full={page.data.full}
      lastUpdate={lastUpdate ?? undefined}
      editOnGithub={{
        owner: "unkeyed",
        repo: "unkey",
        sha: "main",
        path: `apps/engineering/content/docs/${page.file.path}`,
      }}
    >
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription>{page.data.description}</DocsDescription>
      <DocsBody>
        <MDX components={{ ...defaultMdxComponents }} />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return source.generateParams();
}

export function generateMetadata({ params }: { params: { slug?: string[] } }) {
  const page = source.getPage(params.slug);
  if (!page) {
    notFound();
  }

  return {
    title: page.data.title,
    description: page.data.description,
  } satisfies Metadata;
}
