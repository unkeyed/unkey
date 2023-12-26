import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { PageLinks } from "@/components/landing/page-links";
import { allChangelogs } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";

import type { Metadata } from "next";

type Props = {
  params: { date: string };
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
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
    },
    twitter: {
      card: "summary_large_image",
      title: `${changelog?.title} | Unkey`,
      description: changelog?.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
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
          <PageIntro eyebrow={changelog.date} title={changelog.title} centered>
            <p>{changelog.description}</p>
          </PageIntro>
        </header>

        <Container className="mt-24 sm:mt-32 lg:mt-40">
          <FadeIn>
            <div className="prose lg:prose-md mx-auto max-w-5xl">
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
