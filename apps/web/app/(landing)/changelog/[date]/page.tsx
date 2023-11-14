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
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  return {
    title: `${changelog?.title} | Unkey`,
    description: changelog?.description,
    openGraph: {
      title: `${changelog?.title} | Unkey`,
      description: changelog?.description,
      url: `${baseUrl}/changelog/${changelog?.url}`,
      siteName: "unkey.dev",
      images: [
        {
          url: `${baseUrl}/og/changelog?title=${changelog?.title}&date=${changelog?.date}`,
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
