import { Container } from "@/components/container";
import { FadeIn } from "@/components/fade-in";
// import { PageIntro } from "@/components/page-intro";
// import { PageLinks } from "@/components/page-links";
import { notFound } from "next/navigation";

import { CTA } from "@/components/cta";
import { MdxContent } from "@/components/mdx-content";
import { ChangelogLight } from "@/components/svg/changelog";
import { MeteorLines } from "@/components/ui/meteorLines";
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
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";
  const ogUrl = new URL("/og/changelog", baseUrl);

  ogUrl.searchParams.set("title", frontmatter.title ?? "");
  ogUrl.searchParams.set("date", params.date ?? "");

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
      images: [
        {
          url: ogUrl.toString(),
          width: 1200,
          height: 630,
          alt: frontmatter.title,
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title: `${frontmatter?.title} | Unkey`,
      description: frontmatter?.description,
      site: "@unkeydev",
      creator: "@unkeydev",
      images: [
        {
          url: ogUrl.toString(),
          width: 1200,
          height: 630,
          alt: frontmatter.title,
        },
      ],
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

  const _moreChangelogs = await getContentData({
    contentPath: CHANGELOG_PATH,
    filepath: params.date,
  });

  return (
    <>
      <div className="flex flex-col mt-32 text-white/60 mx-auto">
        <div>
          <div className="relative -z-100 max-w-[1000px] mx-auto">
            <ChangelogLight className="w-full" />
          </div>
          <div className="w-full overflow-hidden">
            <MeteorLines number={2} xPos={60} direction="left" />
            <MeteorLines number={2} xPos={200} direction="left" />
            <MeteorLines number={2} xPos={350} direction="left" />
            <MeteorLines number={2} xPos={60} direction="right" />
            <MeteorLines number={2} xPos={200} direction="right" />
            <MeteorLines number={2} xPos={350} direction="right" />
          </div>
        </div>
        <article className="mt-24 sm:mt-32 lg:mt-40">
          <Container className="mt-24 sm:mt-32 lg:mt-40">
            <FadeIn>
              <div className="prose lg:prose-md prose-neutral dark:prose-invert prose-pre:border prose-pre:border-border prose-pre:rounded-lg prose-img:rounded-lg prose-img:border prose-img:border-border mx-auto max-w-5xl ">
                <MdxContent source={serialized} />
              </div>
            </FadeIn>
          </Container>
        </article>
        <CTA />
      </div>
    </>
  );
}
