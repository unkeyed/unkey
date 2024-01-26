import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { PageLinks } from "@/components/landing/page-links";
import { notFound } from "next/navigation";

import { MdxContent } from "@/components/landing/mdx-content";
import { CHANGELOG_PATH, getChangelog, getContentData, getFilePaths } from "@/lib/mdx-helper";
import type { Metadata } from "next";

type Props = {
  params: { date: string };
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { frontmatter } = await getChangelog(params.date);

  if (!frontmatter) {
    return notFound();
  }

  return {
    title: `${frontmatter?.title} | Unkey`,
    description: frontmatter?.description,
    openGraph: {
      title: `${frontmatter?.title} | Unkey`,
      description: frontmatter?.description,
      url: `https://unkey.dev/changelog/${params.date}?title=${encodeURIComponent(
        frontmatter?.title,
      )}`,
      siteName: "unkey.dev",
    },
    twitter: {
      card: "summary_large_image",
      title: `${frontmatter?.title} | Unkey`,
      description: frontmatter?.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
}

export const generateStaticParams = async () => {
  const changelogs = await getFilePaths(CHANGELOG_PATH);
  // Remove file extensions for page paths
  changelogs.map((path) => path.replace(/\.mdx?$/, "")).map((date) => ({ params: { date } }));
  return changelogs;
};

export default async function ChangeLogLayout({
  params,
}: {
  params: { date: string };
}) {
  const { frontmatter, serialized } = await getChangelog(params.date);

  if (!serialized) {
    return notFound();
  }

  const moreChangelogs = await getContentData({
    contentPath: CHANGELOG_PATH,
    filepath: params.date,
  });

  return (
    <>
      <article className="mt-24 sm:mt-32 lg:mt-40">
        <header>
          <PageIntro
            eyebrow={new Date(frontmatter.date).toDateString()}
            title={frontmatter.title}
            centered
          >
            <p>{frontmatter.description}</p>
          </PageIntro>
        </header>

        <Container className="mt-24 sm:mt-32 lg:mt-40">
          <FadeIn>
            <div className="prose lg:prose-md prose-neutral dark:prose-invert prose-pre:border prose-pre:border-border prose-pre:rounded-lg prose-img:rounded-lg prose-img:border prose-img:border-border mx-auto max-w-5xl ">
              <MdxContent source={serialized} />
            </div>
          </FadeIn>
        </Container>
      </article>

      {moreChangelogs.length > 0 && (
        <PageLinks
          className="mt-24 sm:mt-32 lg:mt-40"
          title="Read more changelogs"
          pages={moreChangelogs}
          contentType="changelog"
        />
      )}
    </>
  );
}
