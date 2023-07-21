import Link from "next/link";
import { Border } from "@/components/landing/border";
import { Button } from "@/components/landing/button";
import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { allChangelogs } from "contentlayer/generated";

function Changelog({ changelogs } = { changelogs: allChangelogs }) {
  return (
    <Container className="mt-40">
      <FadeIn>
        <h2 className="text-2xl font-semibold font-display text-neutral-950">Changelogs</h2>
      </FadeIn>
      <div className="mt-10 space-y-20 sm:space-y-24 lg:space-y-32">
        {changelogs.map((changelog) => (
          <FadeIn key={changelog.title}>
            <article>
              <Border className="grid grid-cols-3 pt-16 gap-x-8 gap-y-8">
                <div className="col-span-full sm:flex sm:items-center sm:justify-between sm:gap-x-8 lg:col-span-1 lg:block">
                  <div className="sm:flex sm:items-center sm:gap-x-6 lg:block">
                    <h3 className="mt-6 text-sm font-semibold text-neutral-950 sm:mt-0 lg:mt-8">
                      {changelog.title}
                    </h3>
                  </div>
                  <div className="flex mt-1 gap-x-4 sm:mt-0 lg:block">
                    <p className="text-sm tracking-tight text-neutral-950 after:ml-4 after:font-semibold after:text-neutral-300 after:content-['/'] lg:mt-2 lg:after:hidden" />
                    <p className="text-sm text-neutral-950 lg:mt-2">
                      <time dateTime={changelog.date}>{changelog.date}</time>
                    </p>
                  </div>
                </div>
                <div className="col-span-full lg:col-span-2 lg:max-w-2xl">
                  <p className="text-4xl font-medium font-display text-neutral-950">
                    <Link href={changelog.url}>{changelog.title}</Link>
                  </p>
                  <div className="mt-6 space-y-6 text-base text-neutral-600">
                    {changelog.summary?.map((paragraph) => (
                      <p key={paragraph}>{paragraph}</p>
                    ))}
                  </div>
                  <div className="flex mt-8">
                    <Button
                      href={changelog.url}
                      aria-label={`Read this Changelog for ${changelog.title}`}
                    >
                      Read the Changelog
                    </Button>
                  </div>
                </div>
              </Border>
            </article>
          </FadeIn>
        ))}
      </div>
    </Container>
  );
}

export const metadata = {
  title: "Changelog | Unkey",
  description: "Stay up-to-date with the latest updates and changes to Unkey",
};

export default async function Changelogs() {
  const changelogs = allChangelogs.sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
  );

  return (
    <>
      <PageIntro eyebrow="" title="Unkey Changelog">
        <p>
          We are constantly improving our product, fixing bugs and introducing features. Here you
          can find the latest updates and changes to Unkey.
        </p>
      </PageIntro>

      <Changelog changelogs={changelogs} />
    </>
  );
}
