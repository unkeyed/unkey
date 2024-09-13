import { SuggestedBlogs } from "@/components/blog/suggested-blogs";
import { CTA } from "@/components/cta";
import { SearchInput } from "@/components/glossary/input";
import { MDX } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Label } from "@/components/ui/label";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { allGlossaries } from "content-collections";
import type { Glossary } from "content-collections";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Fragment } from "react";
import { categories } from "../data";
import TermsNavigation from "@/components/glossary/terms-navigation";

export const generateStaticParams = async () =>
  allGlossaries.map((term) => ({
    slug: term.slug,
  }));

export function generateMetadata({
  params,
}: {
  params: { slug: string };
}): Metadata {
  const term = allGlossaries.find((term) => term.slug === `${params.slug}`);
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
  const term = allGlossaries.find((term) => term.slug === `${params.slug}`) as Glossary;
  if (!term) {
    notFound();
  }

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
          <div >
            <h2 className="w-full mb-4 font-semibold text-left blog-heading-gradient">
              Find a term
            </h2>
            <SearchInput
              placeholder="Search"
              className="rounded-lg mb-4 border-[.75px] border-white/20 lg:w-[232px]"
            />
            <TermsNavigation />
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
                    {term.title}
                  </span>
                </div>
                <h1 className="not-prose blog-heading-gradient text-left text-4xl font-medium leading-[56px] tracking-tight sm:text-5xl sm:leading-[72px]">
                  {term.title}
                </h1>
                <p className="mt-8 text-lg font-medium leading-8 not-prose text-white/60 lg:text-xl">
                  {term.description}
                </p>
              </div>
              <div className="mt-12 prose-sm lg:pr-24 md:prose-md text-white/60 sm:mx-6 prose-strong:text-white/90 prose-code:text-white/80 prose-code:bg-white/10 prose-code:px-2 prose-code:py-1 prose-code:border-white/20 prose-code:rounded-md prose-pre:p-0 prose-pre:m-0 prose-pre:leading-6">
                <MDX code={term.mdx} />
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
                        <li key={`#${heading.slug}`}>
                          <Link
                            href={`#${heading.slug}`}
                            className={cn("text-white/60 hover:text-white", {
                              "text-sm": heading.level > 2,
                              "ml-4": heading.level === 3,
                              "ml-8": heading.level === 4,
                            })}
                          >
                            {heading.text}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {/* Related Blogs */}
                <div>
                  <h3 className="text-lg font-semibold text-white mb-4">Related Terms</h3>
                  <SuggestedBlogs currentPostSlug={term.url} />
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
