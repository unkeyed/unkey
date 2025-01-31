import { companySource } from "@/app/source";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/page";
import type { Metadata } from "next";

import { getGithubLastEdit } from "fumadocs-core/server";
import { notFound } from "next/navigation";

export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;

  const page = companySource.getPage(params.slug);

  if (!page) {
    notFound();
  }

  const MDX = page.data.body;

  return (
    <DocsPage
      toc={page.data.toc}
      full={page.data.full}
      tableOfContent={{
        style: "clerk",
        single: true,
      }}
      lastUpdate={
        (await getGithubLastEdit({
          owner: "unkeyed",
          repo: "unkey",
          path: `apps/engineering/content/company/${page.file.path}`,
        })) ?? undefined
      }
    >
      <DocsTitle>{page.data.title}</DocsTitle>

      <DocsDescription className="text-sm">{page.data.description}</DocsDescription>

      <DocsBody className="font-mono text-sm">
        <MDX components={{ ...defaultMdxComponents }} />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return companySource.generateParams();
}

export async function generateMetadata({ params }: { params: Promise<{ slug?: string[] }> }) {
  const page = companySource.getPage((await params).slug);
  if (!page) {
    notFound();
  }

  return {
    title: page.data.title,
    description: page.data.description,
  } satisfies Metadata;
}
