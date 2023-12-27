import { Container } from "@/components/landing/container";
import { ArrowLeft, ExternalLink } from "lucide-react";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import { templates } from "../data";

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
      <div className="relative flex flex-col items-start mt-16 space-y-8 lg:flex-row lg:mt-32 lg:space-y-0 ">
        <div className="self-start w-full px-4 mx-auto lg:sticky top-32 h-max lg:w-2/5 sm:px-6 lg:px-8 ">
          <Link
            href="/templates"
            className="flex items-center gap-1 text-xs duration-200 text-content-subtle hover:text-foreground"
          >
            <ArrowLeft className="w-4 h-4" /> Back to Templates
          </Link>
          <div className="pb-10 mt-4">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-6xl">
              {template.title}
            </h2>
            <p className="mt-2 text-gray-500">{template.description}</p>
          </div>
          <div className="flex items-center justify-between gap-4">
            {template.url ? (
              <Link
                target="_blank"
                className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-gray-900 duration-150 border rounded-md hover:border-gray-900"
                href={`${template.url}?ref=unkey.dev`}
              >
                Website
                <ExternalLink className="inline-block w-3 h-3 ml-1" />
              </Link>
            ) : null}
            <Link
              target="_blank"
              className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-gray-900 duration-150 border rounded-md hover:border-gray-900"
              href={template.repository}
            >
              Repository
              <ExternalLink className="inline-block w-3 h-3 ml-1" />
            </Link>
          </div>

          <dl className="grid grid-cols-2 gap-6 mt-10">
            {Object.entries(tags)
              .filter(([_, value]) => !!value)
              .map(([key, value]) => (
                <div key={key}>
                  <dt className="text-sm font-medium text-gray-900">{value}</dt>
                  <dd className="text-sm text-gray-500 ">{key}</dd>
                </div>
              ))}

            <div>
              <span className="mt-1 text-sm text-gray-500">by </span>
              <span className="text-sm font-medium text-gray-900">
                {template.authors.join(", ")}
              </span>
            </div>
          </dl>
        </div>

        <div className="w-full border-gray-100 lg:border-l lg:pl-8 lg:w-3/5">
          {template.image ? (
            <div className="overflow-hidden bg-gray-100 rounded-lg ">
              <img
                src={template.image}
                alt={template.description}
                className="object-cover object-center w-full h-full"
              />
            </div>
          ) : null}

          <ReactMarkdown
            className="max-w-5xl mx-auto mt-16 prose lg:prose-md"
            remarkPlugins={[remarkGfm]}
            //  @ts-ignore
            rehypePlugins={[rehypeRaw]}
          >
            {readme}
          </ReactMarkdown>
        </div>
      </div>
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
