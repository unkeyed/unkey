import { ArrowLeft } from "lucide-react";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import { templates } from "../data";

import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";
import { ChangelogLight } from "@/components/svg/changelog";
import { TemplateComponents } from "@/components/template/mdx-components";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { Separator } from "@/components/ui/separator";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

type Props = {
  params: {
    slug: string;
  };
};

export const revalidate = 3600; // 1 hour

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
    <>
      <div className="relative mx-auto -z-100 pt-[64px]">
        <ChangelogLight className="w-full max-w-[1000px] mx-auto -top-40" />
      </div>

      <div className="w-full h-full overflow-clip -z-20">
        <MeteorLinesAngular number={1} xPos={0} speed={10} delay={5} className="overflow-hidden" />
        <MeteorLinesAngular number={1} xPos={0} speed={10} delay={0} className="overflow-hidden" />
        <MeteorLinesAngular
          number={1}
          xPos={100}
          speed={10}
          delay={7}
          className="overflow-hidden sm:hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={100}
          speed={10}
          delay={2}
          className="overflow-hidden sm:hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={200}
          speed={10}
          delay={7}
          className="overflow-hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={200}
          speed={10}
          delay={2}
          className="overflow-hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={400}
          speed={10}
          delay={5}
          className="hidden overflow-hidden md:block"
        />
        <MeteorLinesAngular
          number={1}
          xPos={400}
          speed={10}
          delay={0}
          className="hidden overflow-hidden md:block"
        />
      </div>
      <div className="container flex flex-wrap px-8 mx-auto mt-16 text-white/60">
        <div className="flex flex-col self-start w-full px-0 mx-0 xl:w-1/3 xl:sticky top-20">
          <Link
            href="/templates"
            className="flex items-center gap-1 text-sm duration-200 text-white/60 hover:text-white/80"
          >
            <ArrowLeft className="w-4 h-4" /> Back to Templates
          </Link>
          <div className="mb-8 sm:mt-16">
            <h2 className="sm:text-[40px] sm:text-5xl font-medium tracking-tight blog-heading-gradient leading-[56px] md:w-2/3 xl:w-full text-balance">
              {template.title}
            </h2>
            <p className="mt-2 text-base leading-6 sm:mt-6 lg:mt-12 text-white/60">
              {template.description}
            </p>
          </div>
          <div className="flex items-center justify-between gap-4 sm:mt-20">
            {template.url ? (
              <Link
                target="_blank"
                className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-black transition-all duration-200 transform bg-white border border-white rounded-md hover:bg-black hover:text-white max-w-1/2"
                href={`${template.url}?ref=unkey.com`}
              >
                Website
              </Link>
            ) : null}
            <Link
              target="_blank"
              className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-black transition-all duration-200 transform bg-white border border-white rounded-md hover:bg-black hover:text-white max-w-1/2"
              href={template.repository}
            >
              Repository
            </Link>
          </div>

          <div className="grid grid-rows-2 mt-12 ">
            <dl className="flex flex-row w-full my-4">
              <dt className="w-1/2 text-sm text-white/50">Written by </dt>
              <dd className="w-1/2 text-sm font-medium text-white text-end">
                {template.authors.join(", ")}
              </dd>
            </dl>

            {Object.entries(tags)
              .filter(([_, value]) => !!value)
              .map(([key, value]) => (
                <div key={key}>
                  <Separator orientation="horizontal" />
                  <dl className="flex flex-row w-full my-4">
                    <dt className="w-1/2 text-sm text-white/50">{key}</dt>
                    <dl className="w-1/2 text-sm font-medium text-white text-end">{value}</dl>
                  </dl>
                </div>
              ))}
          </div>
        </div>
        <div className="flex flex-col w-full mt-8 mb-24 xl:w-2/3 md:mt-0 xl:pt-24 xl:pl-24 prose-strong:text-white/90 prose-code:text-white/80 prose-code:bg-white/10 prose-code:px-2 prose-code:py-1 prose-code:border-white/20 prose-code:rounded-md">
          <div>
            {template.image ? (
              <Frame size={"sm"} className="">
                <img src={template.image} alt={template.description} />
              </Frame>
            ) : null}
          </div>
          <ReactMarkdown
            className="flex flex-col gap-10 mt-16 xl:px-10 sm:mt-20 sm:mx-4"
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
    </>
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
      url: `https://unkey.com/blog/${params.slug}`,
      siteName: "unkey.com",
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
