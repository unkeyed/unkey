"use client";
import { CTA } from "@/components/cta";
import { ChangelogLight } from "@/components/svg/changelog";

import { type Glossary, allGlossaries } from "@/.content-collections/generated";
import { PrimaryButton } from "@/components/button";
import { Container } from "@/components/container";
import { FilterableCommand } from "@/components/glossary/search";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { LogIn } from "lucide-react";
import { Zap } from "lucide-react";
import Link from "next/link";

export function GlossaryClient() {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ".split("");

  const groupedTerms = allGlossaries.reduce(
    (acc, term) => {
      const firstLetter = term.title[0].toUpperCase();
      if (!acc[firstLetter]) {
        acc[firstLetter] = [];
      }
      acc[firstLetter].push(term);
      return acc;
    },
    {} as Record<string, Array<Glossary>>,
  );

  return (
    <div className="flex flex-col mx-auto py-28 lg:py-16 text-white/60">
      <div>
        <div className="relative -z-100 max-w-[1000px] mx-auto">
          <ChangelogLight className="w-full -top-52" />
        </div>
        <div className="w-full h-full overflow-clip -z-20">
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
            className="overflow-hidden sm:hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={0}
            className="overflow-hidden sm:hidden md:block"
          />
        </div>
      </div>
      <Container className="text-center py-28 space-y-10">
        <div>
          <h1>
            <span className="mt-6 block max-w-5xl font-display text-5xl font-medium tracking-tight blog-heading-gradient [text-wrap:balance] max-sm:text-4xl sm:mt-2 sm:text-6xl mx-auto">
              API Glossary: A comprehensive guide to API terminology by Unkey
            </span>
          </h1>
          <div className="max-w-3xl text-xl text-white/60 mx-auto">
            <p className="mt-6 text-base about-founders-text-gradient ">
              With clear definitions and helpful examples, Unkey's API Glossary is your go-to
              resource for understanding the key concepts and terminology in API development.
            </p>
          </div>
        </div>
        <div className="flex justify-center gap-6">
          <p className="mt-6 text-base about-founders-text-gradient ">Ready to protect your API?</p>
          <Link href="https://app.unkey.com" className="group self-end">
            <PrimaryButton shiny IconLeft={LogIn} label="Get started" className="h-8" />
          </Link>
        </div>
      </Container>
      <div className="container mx-auto mt-18 overflow-hidden text-white">
        <div className="mb-24 grid grid-cols-1 gap-4 md:gap-8 pb-24 lg:grid-cols-[15rem_1fr] xl:grid-cols-[15rem_1fr_15rem]">
          {/* Left Sidebar */}
          <div>
            <h2 className="w-full mb-4 font-semibold text-left blog-heading-gradient">
              Find a term
            </h2>
            <FilterableCommand
              placeholder="Search"
              className="rounded-lg mb-4 border-[.75px] border-white/20 lg:w-[232px]"
              terms={allGlossaries}
            />
          </div>
          <div className="col-span-2">
            <div className="justify-between flex border-b border-white/10 pb-8 mb-8">
              {alphabet.map((letter) =>
                groupedTerms[letter]?.length > 0 ? (
                  <Link key={letter} href={`#${letter}`} className="rounded hover:underline">
                    {letter}
                  </Link>
                ) : (
                  <span key={letter} className="rounded text-white/30">
                    {letter}
                  </span>
                ),
              )}
            </div>
            {Object.entries(groupedTerms).map(
              ([letter, letterTerms]) =>
                letterTerms.length > 0 && (
                  <section key={letter} id={letter} className="mb-8 scroll-mt-32">
                    <h2 className="text-2xl font-semibold mb-4 grid-cols-1">{letter}</h2>
                    <div className="grid grid-cols-1 gap-8 auto-rows-fr xl:grid-cols-3 md:grid-cols-2 grid-col-1">
                      {letterTerms.map(({ slug, categories, takeaways, term }) => (
                        <Link
                          key={slug}
                          href={`/glossary/${slug}`}
                          className="flex flex-col items-start justify-between h-full overflow-hidden duration-200 border rounded-xl border-white/10 hover:border-white/20"
                        >
                          <div className="relative w-full h-full">
                            <div className="p-4 rounded-md space-y-2 bg-gradient-to-br from-[rgb(22,22,22)] to-[rgb(0,0,0)] border-b border-white/10">
                              <div className="p-4 rounded-md space-y-2 ">
                                <h3 className="text-sm font-semibold flex items-center text-white">
                                  <Zap className="mr-2 h-5 w-5" /> TL;DR
                                </h3>
                                <p className="text-sm text-white/80 ">{takeaways.tldr}</p>
                              </div>
                            </div>
                          </div>
                          <div className="flex flex-col justify-start w-full h-full p-4">
                            <div>
                              <div className="flex flex-row justify-start w-full h-full gap-3">
                                {categories.length > 0
                                  ? categories.map((categorySlug) => (
                                      <div
                                        key={categorySlug}
                                        className="px-2 py-1 text-xs rounded-md bg-[rgb(26,26,26)] text-white/60"
                                      >
                                        {categorySlug // unslugged
                                          .replace(/-/g, " ")
                                          .replace(/\b\w/g, (char) => char.toUpperCase())}
                                      </div>
                                    ))
                                  : null}
                              </div>
                            </div>
                            <div className="flex flex-col items-end content-end justify-end w-full h-full">
                              <div className="w-full h-12 mt-6">
                                <h3 className="text-lg font-semibold leading-6 text-left text-white group-hover:text-gray-600 line-clamp-2">
                                  {term}
                                </h3>
                              </div>
                            </div>
                          </div>
                        </Link>
                      ))}
                    </div>
                  </section>
                ),
            )}
          </div>
        </div>
      </div>
      <CTA />
    </div>
  );
}

// <Link href={`/glossary/${slug}`} key={slug} className="block">
//   <Card className="w-full bg-white/5 shadow-[0_0_10px_rgba(255,255,255,0.1)] rounded-xl overflow-hidden relative border-white/20">
//     <CardHeader>
// <Frame size="sm">
//   <div className="p-4 rounded-md space-y-2">
//     <h3 className="text-sm font-semibold flex items-center text-white">
//       <Zap className="mr-2 h-5 w-5" /> TL;DR
//     </h3>
//     <p className="text-sm text-white/80">{takeaways.tldr}</p>
//   </div>
// </Frame>
//     </CardHeader>
//     <CardContent>
//       <h4 className="text-md font-semibold text-white mb-2">{term}</h4>
//     </CardContent>
//   </Card>
// </Link>
