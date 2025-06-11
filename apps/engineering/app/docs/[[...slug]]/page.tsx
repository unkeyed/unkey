import { source } from "@/app/source";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/page";
import type { Metadata } from "next";

import { getGithubLastEdit } from "fumadocs-core/server";
import { notFound } from "next/navigation";

export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;

  const page = source.getPage(params.slug);

  if (!page) {
    notFound();
  }  

   if (page.slugs.length === 0) {
    return (
      <div className="min-h-screen border text-center -mt-16 pt-16 flex items-center w-screen justify-center ">
        <div>
          <h1 className="text-7xl md:text-8xl font-bold  leading-none  uppercase tracking-tight">
            Docs
          </h1>
          <p className="text-xl mt-8 font-light ">Check the sidebar</p>
        </div>
      </div>
    );
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

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug?: string[] }>;
}) {
  const page = source.getPage((await params).slug);
  if (!page) {
    notFound();
  }

  return {
    title: page.data.title,
    description: page.data.description,
  } satisfies Metadata;
}
