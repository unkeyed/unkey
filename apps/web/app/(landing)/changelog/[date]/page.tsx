import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { PageLinks } from "@/components/landing/page-links";
import { allChangelogs } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";

import type { Metadata } from 'next'
 
type Props = {
  params: { date: string }
}
 
export async function generateMetadata(
  { params }: Props,
): Promise<Metadata> {
  // read route params
  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date,
  );

 
  return {
    title: `${changelog?.title} | Unkey`,
  description: changelog?.description,
  openGraph: {
    title: `${changelog?.title} | Unkey`,
  description: changelog?.description,
    url: `https://unkey.dev/changelog/${changelog?.url}`,
    siteName: "unkey.dev",
    images: [
      {
        url: `https://unkey.dev/og?title=${changelog?.title}`,
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: `${changelog?.title} | Unkey`,
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/unkey.png",
  },
  }
}


export const generateStaticParams = async () =>
  allChangelogs.map((c) => ({
    date: new Date(c.date).toISOString().split("T")[0],
  }));

export default async function ChangeLogLayout({
  params,
}: {
  params: { date: string };
}) {
  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date,
  );

  if (!changelog?.body) {
    return notFound();
  }

  const Content = getMDXComponent(changelog.body.code);

  const moreChangelogs = allChangelogs.filter((p) => p.date !== params.date).slice(0, 2);

  return (
    <>
      <article className="mt-24 sm:mt-32 lg:mt-40">
        <header>
          <PageIntro eyebrow="" title={changelog.title} centered>
            <p>{changelog.description}</p>
          </PageIntro>

          <div className="mt-24 border-t border-neutral-200 bg-white/50 sm:mt-32 lg:mt-40">
            <Container>
              <div className="max-w-5xl mx-auto">
                <dl className="grid grid-cols-1 -mx-6 text-sm text-neutral-950 sm:mx-0 sm:grid-cols-3">
                  <div className="px-6 py-4 border-t border-neutral-200 first:border-t-0 sm:border-l sm:border-t-0">
                    <dt className="font-semibold">Number of changes</dt>
                    <dd>{changelog.changes}</dd>
                  </div>
                  <div className="px-6 py-4 border-t border-neutral-200 first:border-t-0 sm:border-l sm:border-t-0">
                    <dt className="font-semibold">Date</dt>
                    <dd>
                      <time dateTime={changelog.date}>{changelog.date}</time>
                    </dd>
                  </div>
                  <div className="px-6 py-4 border-t border-neutral-200 first:border-t-0 sm:border-l sm:border-t-0">
                    <dt className="font-semibold">New Features</dt>
                    <dd>{changelog.features}</dd>
                  </div>
                </dl>
              </div>
            </Container>
          </div>

          <div className="border-y border-neutral-200 bg-neutral-100">
            <div className="-my-px mx-auto max-w-[76rem] bg-neutral-200" />
          </div>
        </header>

        <Container className="mt-24 sm:mt-32 lg:mt-40">
          <FadeIn>
            <div className="max-w-5xl mx-auto prose lg:prose-md">
              <Content />
            </div>
          </FadeIn>
        </Container>
      </article>

      {moreChangelogs.length > 0 && (
        <PageLinks
          className="mt-24 sm:mt-32 lg:mt-40"
          title="Read more changelogs"
          pages={moreChangelogs}
        />
      )}
    </>
  );
}
