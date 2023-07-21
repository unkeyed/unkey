import Link from "next/link";
import clsx from "clsx";

import { Border } from "@/components/landing-components/border";
import { Container } from "@/components/landing-components/container";
import { FadeIn, FadeInStagger } from "@/components/landing-components/fade-in";
import { GridPattern } from "@/components/landing-components/grid-pattern";
import { SectionIntro } from "@/components/landing-components/section-intro";
import { formatDate } from "@/lib/formatDate";

interface Page {
  title: string;
  description: string;
  date: string;
  url: string;
}

function ArrowIcon(props: any) {
  return (
    <svg viewBox="0 0 24 6" aria-hidden="true" {...props}>
      <path fillRule="evenodd" clipRule="evenodd" d="M24 3 18 .5v2H0v1h18v2L24 3Z" />
    </svg>
  );
}

function PageLink({ page }: { page: Page }) {
  return (
    <article key={page.url}>
      <Border position="left" className="relative flex flex-col items-start pl-8">
        <h3 className="mt-6 text-base font-semibold text-neutral-950">{page.title}</h3>
        <time dateTime={page.date} className="order-first text-sm text-neutral-600">
          {formatDate(page.date)}
        </time>
        <p className="mt-2.5 text-base text-neutral-600">{page.description}</p>
        <Link
          href={page.url}
          className="mt-6 flex gap-x-3 text-base font-semibold text-neutral-950 transition hover:text-neutral-700"
          aria-label={`Read more: ${page.title}`}
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
  className,
}: {
  title: string;
  intro?: string;
  pages: Page[];
  className?: string;
}) {
  return (
    <div className={clsx("relative pt-24 sm:pt-32 lg:pt-40", className)}>
      <div className="absolute inset-x-0 top-0 -z-10 h-[884px] overflow-hidden rounded-t-4xl bg-gradient-to-b from-neutral-50">
        <GridPattern
          className="absolute inset-0 h-full w-full fill-neutral-100 stroke-neutral-950/5 [mask-image:linear-gradient(to_bottom_left,white_40%,transparent_50%)]"
          yOffset={-270}
        />
      </div>

      <SectionIntro title={title} smaller>
        {intro && <p>{intro}</p>}
      </SectionIntro>

      <Container className={intro ? "mt-24" : "mt-16"}>
        <FadeInStagger className="grid grid-cols-1 gap-x-8 gap-y-16 lg:grid-cols-2">
          {pages.map((page) => (
            <FadeIn key={page.url}>
              <PageLink page={page} />
            </FadeIn>
          ))}
        </FadeInStagger>
      </Container>
    </div>
  );
}
