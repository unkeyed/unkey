import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";

import { FilterableCommand } from "@/components/glossary/search";
import TermsRolodexDesktop from "@/components/glossary/terms-rolodex-desktop";
import TermsStepperMobile from "@/components/glossary/terms-stepper-mobile";
import { MDX } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { cn } from "@/lib/utils";
import { allGlossaries } from "content-collections";
import { Zap } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { FAQ } from "./faq";
import Takeaways from "./takeaways";

export const generateStaticParams = async () =>
  allGlossaries.map((term) => ({
    slug: term.slug,
  }));

export function generateMetadata({
  params,
}: {
  params: { slug: string };
}): Metadata {
  const term = allGlossaries.find((term) => term.slug === params.slug);
  if (!term) {
    notFound();
  }
  return {
    title: `${term.title} | Unkey Glossary`,
    description: term.description,
    openGraph: {
      title: `${term.title} | Unkey Glossary`,
      description: term.description,
      url: `https://unkey.com/glossary/${term.slug}`,
      siteName: "unkey.com",
      type: "article",
    },
    twitter: {
      card: "summary_large_image",
      title: `${term.title} | Unkey Glossary`,
      description: term.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
}

const GlossaryTermWrapper = async ({ params }: { params: { slug: string } }) => {
  const term = allGlossaries.find((term) => term.slug === params.slug);
  if (!term) {
    notFound();
  }

  const relatedTerms: {
    slug: string;
    term: string;
    tldr: string;
  }[] = [];
  return (
    <>
      <div className="container pt-48 mx-auto sm:overflow-hidden md:overflow-visible scroll-smooth">
        <div>
          <TopLeftShiningLight className="hidden h-full -z-40 sm:block" />
        </div>
        <div className="w-full h-full overflow-hidden -z-20">
          <MeteorLinesAngular
            number={1}
            xPos={0}
            speed={10}
            delay={5}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={0}
            speed={10}
            delay={0}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={7}
            className="overflow-hidden md:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={2}
            className="overflow-hidden md:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={7}
            className="hidden overflow-hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={2}
            className="hidden overflow-hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={5}
            className="hidden overflow-hidden lg:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={0}
            className="hidden overflow-hidden lg:block"
          />
        </div>
        <div className="overflow-hidden -z-40">
          <TopRightShiningLight />
        </div>
        <div className="w-full">
          <div className="mb-24 grid grid-cols-1 gap-4 md:gap-8 pb-24 lg:grid-cols-[15rem_1fr] xl:grid-cols-[15rem_1fr_15rem]">
            {/* Left Sidebar */}
            <div className="block">
              <div className="sticky top-24 flex flex-col">
                <p className="w-full mb-4 font-semibold text-left blog-heading-gradient">
                  Find a term
                </p>
                <FilterableCommand
                  placeholder="Search"
                  className="rounded-lg mb-4 border-[.75px] border-white/20 lg:w-[232px]"
                  terms={allGlossaries}
                />
                <div className="flex-grow">
                  <p className="w-full my-4 font-semibold text-left blog-heading-gradient">Terms</p>
                  <TermsRolodexDesktop
                    className="flex-grow hidden lg:block"
                    terms={allGlossaries.map((term) => ({ slug: term.slug, title: term.title }))}
                  />
                  <TermsStepperMobile
                    className="flex-grow lg:hidden"
                    terms={allGlossaries.map((term) => ({ slug: term.slug, title: term.title }))}
                  />
                </div>
              </div>
            </div>
            {/* Main Content */}
            <div>
              <div className="prose sm:prose-sm md:prose-md sm:mx-6">
                <div className="flex items-center gap-5 p-0 m-0 mb-8 text-xl font-medium leading-8">
                  <Link href="/glossary">
                    <span className="text-transparent bg-gradient-to-r bg-clip-text from-white to-white/60">
                      Glossary
                    </span>
                  </Link>
                  <span className="text-white/40">/</span>
                  <span className="text-transparent capitalize bg-gradient-to-r bg-clip-text from-white to-white/60">
                    {term.term}
                  </span>
                </div>
                <h1 className="not-prose blog-heading-gradient text-left font-medium tracking-tight text-4xl">
                  {term.h1}
                </h1>
                <p className="mt-8 text-lg font-medium leading-8 not-prose text-white/60 lg:text-xl">
                  {term.intro}
                </p>
              </div>
              <div className="mt-12 sm:mx-6">
                <Takeaways takeaways={term.takeaways} term={term.term} />
              </div>
              <div className="mt-12 prose-sm md:prose-md text-white/60 sm:mx-6 prose-strong:text-white/90 prose-code:text-white/80 prose-code:bg-white/10 prose-code:px-2 prose-code:py-1 prose-code:border-white/20 prose-code:rounded-md prose-pre:p-0 prose-pre:m-0 prose-pre:leading-6">
                <MDX code={term.mdx} />
              </div>
              <div className="mt-12 sm:mx-6">
                <FAQ
                  items={[
                    {
                      // provide some FAQs for questions & answers about mime types:
                      question: "What is a mime type?",
                      answer:
                        "A mime type is a standard way to describe the format of a file. It is used to identify the type of data contained in a file, such as an image, a video, or a document. Mime types are essential for web browsers to correctly display and process different types of files.",
                    },
                    {
                      question: "What is the difference between a mime type and a file extension?",
                      answer:
                        "A mime type is a standard way to describe the format of a file. It is used to identify the type of data contained in a file, such as an image, a video, or a document. Mime types are essential for web browsers to correctly display and process different types of files. A file extension is a suffix added to a file name to indicate its type. It is used to help users identify the type of file and to help applications determine how to open or process the file.",
                    },
                    {
                      question: "Which mime types are supported by Unkey?",
                      answer:
                        "Unkey supports a wide range of mime types, including text, image, audio, video, and application-specific types. The full list of supported mime types can be found in the Unkey documentation.",
                    },
                  ]}
                  title={`Questions & Answers about ${term.term}`}
                  description={`We answer common questions about ${term.term}.`}
                  epigraph="FAQ"
                />
              </div>
            </div>
            {/* Right Sidebar */}
            <div className="hidden xl:block">
              <div className="sticky top-24 space-y-8">
                {term.tableOfContents?.length !== 0 && (
                  <div className="not-prose">
                    <h3 className="text-lg font-semibold text-white mb-4">Contents</h3>
                    <ul className="space-y-2">
                      {term.tableOfContents.map((heading) => (
                        <li key={`#${heading?.slug}`}>
                          <Link
                            href={`#${heading?.slug}`}
                            className={cn("text-white/60 hover:text-white", {
                              "text-sm": heading?.level && heading.level > 2,
                              "ml-4": heading?.level && heading.level === 3,
                              "ml-8": heading?.level && heading.level === 4,
                            })}
                          >
                            {heading?.text}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {/* Related Blogs */}
                <div>
                  <h3 className="text-lg font-semibold text-white mb-4">Related Terms</h3>
                  <div className="flex flex-col gap-4">
                    {relatedTerms.length > 0 ? (
                      relatedTerms.map((relatedTerm) => (
                        <Link
                          href={`/glossary/${relatedTerm.slug}`}
                          key={relatedTerm.slug}
                          className="block"
                        >
                          <Card className="w-full bg-white/5 shadow-[0_0_10px_rgba(255,255,255,0.1)] rounded-xl overflow-hidden relative border-white/20">
                            <CardHeader>
                              <Frame size="sm">
                                <div className="p-4 rounded-md space-y-2">
                                  <h3 className="text-sm font-semibold flex items-center text-white">
                                    <Zap className="mr-2 h-5 w-5" /> TL;DR
                                  </h3>
                                  <p className="text-sm text-white/80">{relatedTerm.tldr}</p>
                                </div>
                              </Frame>
                            </CardHeader>
                            <CardContent>
                              <h4 className="text-md font-semibold text-white mb-2">
                                {relatedTerm.term}
                              </h4>
                            </CardContent>
                          </Card>
                        </Link>
                      ))
                    ) : (
                      <p className="text-sm text-white/50">No related terms found.</p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
          <CTA />
        </div>
      </div>
    </>
  );
};

export default GlossaryTermWrapper;
