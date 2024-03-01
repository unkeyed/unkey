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
    <>
      <div className="relative mx-auto -z-100">
        <ChangelogLight className="w-full max-w-[1000px] mx-auto" />
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
      <div className="flex flex-wrap mt-32 text-white/60 mx-auto px-8 sm:px-4 xxl:px-12 xxl:max-w-[1440px]">
        <div className="flex flex-col self-start w-full px-0 mx-0 xxl:w-1/4 xxl:sticky">
          <Link
            href="/templates"
            className="flex items-center gap-1 text-xs duration-200 text-white/60 hover:text-white/80"
          >
            <ArrowLeft className="w-4 h-4" /> Back to Templates
          </Link>
          <div className="mb-8 sm:mt-16">
            <h2 className="sm:text-[40px] sm:text-5xl font-medium tracking-tight blog-heading-gradient leading-[56px] md:w-2/3 xxl:w-full text-balance">
              {template.title}
            </h2>
            <p className="mt-2 text-base leading-6 sm:mt-6 lg:mt-12 text-white/60">
              {template.description}
            </p>
          </div>
          <div className="flex items-center justify-between gap-6 sm:mt-20">
            {template.url ? (
              <Link
                target="_blank"
                className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-black duration-150 bg-white border rounded-md max-w-1/2"
                href={`${template.url}?ref=unkey.dev`}
              >
                Website
              </Link>
            ) : null}
            <Link
              target="_blank"
              className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center duration-150 border rounded-md text-white/60 border-white/10 max-w-1/2"
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
                <div key={key}>
                  <Separator orientation="horizontal" />
                  <div className="flex flex-row w-full my-4">
                    <dd className="w-1/2 text-sm text-white/30">{key}</dd>
                    <dt className="w-1/2 text-sm font-medium text-white text-end">{value}</dt>
                  </div>
                </div>
              ))}
          </dl>
        </div>
        {/* <Separator orientation="horizontal" /> */}
        <div className="flex flex-col w-full mb-24 xxl:w-3/4 sm:mt-20 md:mt-0 xl:pt-24 xxl:pl-24">
          <div>
            {template.image ? (
              <Frame size={"lg"} className="">
                <img src={template.image} alt={template.description} />
              </Frame>
            ) : null}
          </div>
          <ReactMarkdown
            className="flex flex-col gap-10 mt-16 xxl:px-10 sm:mt-20 sm:mx-4"
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
