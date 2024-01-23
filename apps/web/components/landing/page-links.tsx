import clsx from "clsx";
import Link from "next/link";

import { Border } from "@/components/landing/border";
import { Container } from "@/components/landing/container";
import { FadeIn, FadeInStagger } from "@/components/landing/fade-in";
import { GridPattern } from "@/components/landing/grid-pattern";
import { SectionIntro } from "@/components/landing/section-intro";

interface Page {
  frontmatter: Frontmatter;
  slug: string;
}

type Frontmatter = {
  title: string;
  date: string;
  description: string;
  author: string;
};

function ArrowIcon(props: any) {
  return (
    <svg viewBox="0 0 24 6" aria-hidden="true" {...props}>
      <path fillRule="evenodd" clipRule="evenodd" d="M24 3 18 .5v2H0v1h18v2L24 3Z" />
    </svg>
  );
}

function PageLink({ page, contentType }: { page: Page; contentType: string }) {
  return (
    <article key={page.slug}>
      <Border position="left" className="relative flex flex-col items-start pl-8">
        <h3 className="mt-6 text-base font-semibold text-gray-950">{page.frontmatter.title}</h3>
        <time dateTime={page.frontmatter.date} className="order-first text-sm text-gray-600">
          {new Date(page.frontmatter.date).toDateString()}
        </time>
        <p className="mt-2.5 text-base text-gray-600">{page.frontmatter.description}</p>
        <Link
          href={`/${contentType}/${page.slug}`}
          className="mt-6 flex gap-x-3 text-base font-semibold text-gray-950 transition hover:text-gray-700"
          aria-label={`Read more: ${page.frontmatter.title}`}
        >
          Read more
          <ArrowIcon className="w-6 flex-none fill-current" />
          <span className="absolute inset-0" />
        </Link>
      </Border>
    </article>
  );
}

export function PageLinks({
  title,
  intro,
  pages,
  contentType,
  className,
}: {
  title: string;
  intro?: string;
  pages: Page[];
  contentType: string;
  className?: string;
}) {
  return (
    <div className={clsx("relative pt-24 sm:pt-32 lg:pt-40", className)}>
      <div className="rounded-t-4xl absolute inset-x-0 top-0 -z-10 h-[884px] overflow-hidden bg-gradient-to-b from-gray-50">
        <GridPattern
          className="absolute inset-0 h-full w-full fill-gray-100 stroke-gray-950/5 [mask-image:linear-gradient(to_bottom_left,white_40%,transparent_50%)]"
          yOffset={-270}
        />
      </div>

      <SectionIntro title={title} smaller>
        {intro && <p>{intro}</p>}
      </SectionIntro>

      <Container className={intro ? "mt-24" : "mt-16"}>
        <FadeInStagger className="grid grid-cols-1 gap-x-8 gap-y-16 lg:grid-cols-2">
          {pages.map((page) => (
            <FadeIn key={page.slug}>
              <PageLink contentType={contentType} page={page} />
            </FadeIn>
          ))}
        </FadeInStagger>
      </Container>
    </div>
  );
}
