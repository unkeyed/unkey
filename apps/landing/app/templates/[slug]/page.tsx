import { Container } from "@/components/container";
import { ArrowLeft } from "lucide-react";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import { templates } from "../data";

import { CTA } from "@/components/cta";
import { TemplateComponents } from "@/components/template/mdx-components";
import { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

type Props = {
  params: {
    slug: string;
  };
};

export const revalidate = 300; // 5min

export async function generateStaticParams() {
  return Object.keys(templates).map((slug) => ({
    slug,
  }));
}
export default async function Templates(props: Props) {
  const template = templates[props.params.slug];
  if (!template) {
    return notFound();
  }

  const tags: Record<string, string | undefined> = {
    Framework: template.framework,
    Language: template.language,
  };

  const readme = await fetch(template.readmeUrl).then((res) => res.text());
  return (
    <Container>
      <div className="relative flex flex-col items-start mt-16 space-y-8 lg:flex-row lg:mt-32 lg:space-y-0 mb-24">
        <div className="self-start w-full px-4 mx-auto lg:sticky top-32 h-max lg:w-2/5 sm:px-6 lg:px-8 ">
          <Link
            href="/templates"
            className="flex items-center gap-1 text-xs duration-200 text-white/60 hover:text-white/80"
          >
            <ArrowLeft className="w-4 h-4" /> Back to Templates
          </Link>
          <div className="pb-10 mt-4">
            <h2 className="text-3xl font-bold tracking-tight blog-heading-gradient">
              {template.title}
            </h2>
            <p className="mt-2 text-white/60">{template.description}</p>
          </div>
          <div className="flex items-center justify-between gap-4">
            {template.url ? (
              <Link
                target="_blank"
                className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-black duration-150 border rounded-md bg-white"
                href={`${template.url}?ref=unkey.dev`}
              >
                Website
              </Link>
            ) : null}
            <Link
              target="_blank"
              className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-white/60 duration-150 border rounded-md hover:border-white/10"
              href={template.repository}
            >
              Repository
            </Link>
          </div>

          <dl className="grid grid-rows-2 gap-12 mt-12 lg:w-3/9">
            <div className="flex flex-row w-full">
              <span className="text-sm text-white/60 w-1/2">Written by </span>
              <span className="text-sm font-medium text-white text-end w-1/2">
                {template.authors.join(", ")}
              </span>
            </div>
            {Object.entries(tags)
              .filter(([_, value]) => !!value)
              .map(([key, value]) => (
                <div key={key} className="flex flex-row w-full">
                  <dd className="text-sm text-white/60 w-1/2">{key}</dd>
                  <dt className="text-sm font-medium text-white text-end w-1/2">{value}</dt>
                </div>
              ))}
          </dl>
        </div>

        <div className="w-full lg:pl-8 lg:w-6/9">
          {template.image ? (
            <div className="rounded-[30.5px] bg-gradient-to-b from-white/0 to-white/10 p-[.75px] overflow-hidden">
              <div className="p-2 bg-gradient-to-r from-white/10 to-white/20">
                <div className="rounded-[24px] bg-gradient-to-b from-white/20 to-white/10 p-[.75px] overflow-hidden">
                  <img src={template.image} alt={template.description} />
                </div>
              </div>
            </div>
          ) : null}

          <ReactMarkdown
            className="max-w-5xl mx-auto mt-16 prose lg:prose-md"
            remarkPlugins={[remarkGfm]}
            //  @ts-ignore
            rehypePlugins={[rehypeRaw]}
            components={TemplateComponents}
          >
            {readme}
          </ReactMarkdown>
        </div>
      </div>
      <CTA />
    </Container>
  );
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  // read route params
  const template = templates[params.slug];

  return {
    title: `${template?.title} | Unkey`,
    description: template?.description,
    openGraph: {
      title: `${template?.title} | Unkey`,
      description: template?.description,
      url: `https://unkey.dev/blog/${params.slug}`,
      siteName: "unkey.dev",
    },
    twitter: {
      card: "summary_large_image",
      title: `${template?.title} | Unkey`,
      description: template?.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
}
