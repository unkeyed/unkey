"use client";
import { componentSource } from "@/app/source";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/page";
//import { createTypeTable } from 'fumadocs-typescript/ui';

import { notFound } from "next/navigation";
export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;

  const page = componentSource.getPage(params.slug);

  if (!page) {
    notFound();
  }
  //const { AutoTypeTable } = createTypeTable();

  const MDX = page.data.body;

  return (
    <DocsPage toc={page.data.toc} full={page.data.full}>
      <DocsTitle>{page.data.title}</DocsTitle>

      <DocsDescription>{page.data.description}</DocsDescription>

      <DocsBody>
        <MDX components={{ ...defaultMdxComponents }} />
      </DocsBody>
    </DocsPage>
  );
}
/*
export async function generateStaticParams() {
  return componentSource.generateParams();
}

export function generateMetadata({ params }: { params: { slug?: string[] } }) {
  const page = componentSource.getPage(params.slug);
  if (!page) {
    notFound();
  }

  return {
    title: page.data.title,
    description: page.data.description,
  } satisfies Metadata;
}
*/
