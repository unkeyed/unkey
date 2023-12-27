import { Border } from "@/components/landing/border";
import { Button } from "@/components/landing/button";
import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { allChangelogs } from "contentlayer/generated";
import Link from "next/link";

function Changelog({ changelogs } = { changelogs: allChangelogs }) {
  return (
    <Container className="mt-40">
      <FadeIn>
        <h2 className="font-display text-2xl font-semibold text-gray-950">Changelogs</h2>
      </FadeIn>
      <div className="mt-10 space-y-20 sm:space-y-24 lg:space-y-32">
        {changelogs.map((changelog) => (
          <FadeIn key={changelog.title}>
            <article>
              <Border className="grid grid-cols-3 gap-x-8 gap-y-8 pt-16">
                <div className="col-span-full sm:flex sm:items-center sm:justify-between sm:gap-x-8 lg:col-span-1 lg:block">
                  <div className="sm:flex sm:items-center sm:gap-x-6 lg:block">
                    <p className="mt-6 text-sm font-semibold text-gray-950 sm:mt-0 lg:mt-8">
                      {changelog.title}
                    </p>
                  </div>
                  <div className="mt-1 flex gap-x-4 sm:mt-0 lg:block">
                    <p className="text-sm tracking-tight text-gray-950 after:ml-4 after:font-semibold after:text-gray-300 after:content-['/'] lg:mt-2 lg:after:hidden" />
                    <p className="text-sm text-gray-950 lg:mt-2">
                      <time dateTime={changelog.date}>{changelog.date}</time>
                    </p>
                  </div>
                </div>
                <div className="col-span-full lg:col-span-2 lg:max-w-2xl">
                  <h3 className="font-display text-4xl font-medium text-gray-950">
                    <Link href={changelog.url}>{changelog.title}</Link>
                  </h3>
                  <div className="mt-8 flex">
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
  openGraph: {
    title: "Changelog | Unkey",
    description: "Stay up-to-date with the latest updates and changes to Unkey",
    url: "https://unkey.dev/changelog",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/og/changelog",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Changelog | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
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
