import { rfcSource } from "@/app/source";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/page";
import type { Metadata } from "next";
import { LocalDate } from "./local-date";

import { notFound } from "next/navigation";

export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;

  const page = rfcSource.getPage(params.slug);

  if (!page) {
    notFound();
  }

  if (page.slugs.length === 0) {
    return (
      <div className="min-h-screen border text-center -mt-16 pt-16 flex items-center w-screen justify-center ">
        <div>
          <h1 className="text-7xl md:text-8xl font-bold  leading-none  uppercase tracking-tight">
            RFCS
          </h1>
          <p className="text-xl mt-8 font-light ">Check the sidebar</p>
        </div>
      </div>
    );
  }

  const MDX = page.data.body;

  return (
    <DocsPage toc={page.data.toc} full={page.data.full}>
      <DocsTitle>{page.data.title}</DocsTitle>

      <div className="flex flex-col gap-2">
        <div className="flex justify-between items-center font-mono text-sm">
          <span>{page.data.authors.length > 1 ? "Authors" : "Author"}</span>
          <span>{page.data.authors.join(", ")}</span>
        </div>

        <div className="flex justify-between items-center font-mono text-sm">
          <span>Date</span>
          <LocalDate date={new Date(page.data.date)} />
        </div>
      </div>
      <DocsDescription className="text-sm">{page.data.description}</DocsDescription>

      <DocsBody className="font-mono text-sm">
        <MDX components={{ ...defaultMdxComponents }} />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return rfcSource.generateParams();
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug?: string[] }>;
}) {
  const page = rfcSource.getPage((await params).slug);
  if (!page) {
    notFound();
  }

  return {
    title: page.data.title,
    description: page.data.description,
  } satisfies Metadata;
}
