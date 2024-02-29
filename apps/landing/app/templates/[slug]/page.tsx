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
    <div className="flex flex-col xl:max-w-[1440px] mx-6 md:mx-20 xl:mx-auto lg:mx-auto lg:px-10">
      {/* <div> */}
      {/* <div className="relative -z-100 max-w-[1000px] mx-auto">
          <ChangelogLight className="w-full top-40" />
        </div>
        <div className="w-full overflow-clip">
          <MeteorLinesAngular number={2} xPos={0} />
          <MeteorLinesAngular number={2} xPos={200} />
          <MeteorLinesAngular number={2} xPos={400} />
        </div>
      </div> */}
      <div className="flex flex-col xl:flex-row items-start mt-16 lg:mt-32 lg:space-y-0 mb-24 gap-12">
        <div className="self-start w-full mx-auto xl:sticky xl:w-1/4">
          <Link
            href="/templates"
            className="flex items-center gap-1 text-xs duration-200 text-white/60 hover:text-white/80"
          >
            <ArrowLeft className="w-4 h-4" /> Back to Templates
          </Link>
          <div className="mb-8 xxs:mt-16">
            <h2 className="xxs:text-[40px] xs:text-5xl font-medium tracking-tight blog-heading-gradient leading-[56px] md:w-2/3 xl:w-full text-balance">
              {template.title}
            </h2>
            <p className="xxs:mt-6 lg:mt-12 mt-2 text-white/60 text-base leading-6">
              {template.description}
            </p>
          </div>
          <div className="flex items-center justify-between gap-6 sm:mt-20">
            {template.url ? (
              <Link
                target="_blank"
                className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-black duration-150 border rounded-md bg-white max-w-1/2"
                href={`${template.url}?ref=unkey.dev`}
              >
                Website
              </Link>
            ) : null}
            <Link
              target="_blank"
              className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-white/60 duration-150 border rounded-md border-white/10  max-w-1/2"
              href={template.repository}
            >
              Repository
            </Link>
          </div>

          <dl className="grid grid-rows-2 mt-12 ">
            <div className="flex flex-row w-full h-fit">
              <span className="text-[13px] text-white/30 w-1/2">Written by </span>
              <span className="text-[15px] font-medium text-white text-end w-1/2">
                {template.authors.join(", ")}
              </span>
            </div>

            {Object.entries(tags)
              .filter(([_, value]) => !!value)
              .map(([key, value]) => (
                <div>
                  <Separator orientation="horizontal" />
                  <div key={key} className="flex flex-row w-full my-4">
                    <dd className="text-sm text-white/30 w-1/2">{key}</dd>
                    <dt className="text-sm font-medium text-white text-end w-1/2">{value}</dt>
                  </div>
                </div>
              ))}
          </dl>
          <Separator orientation="horizontal" />
        </div>

        <div className="w-full xxs:mt-12 sm:mt-20 md:mt-0 lg:pt-24 xl:w-2/3 ">
          <div>
            {template.image ? (
              <Frame size={"lg"} className="xl:ml-10">
                <img src={template.image} alt={template.description} />
              </Frame>
            ) : null}
          </div>
          <ReactMarkdown
            className="max-w-5xl mx-auto px-0 xxs:mt-20 mt-16 flex flex-col gap-10 xxs:mx-4"
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
    </div>
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
