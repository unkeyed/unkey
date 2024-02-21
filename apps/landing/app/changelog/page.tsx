import { Container } from "@/components/container";
import { CTA } from "@/components/cta";
import { FadeIn } from "@/components/fade-in";
import { Frontmatter, Tags } from "@/lib/mdx-helper";
import { CHANGELOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import { ChangelogGrid } from "./changelog-grid";

type Changelog = {
  frontmatter: Frontmatter;
  slug: string;
};

type ChangelogsType = Changelog[];

function Changelog({ changelogs }: { changelogs: ChangelogsType }) {
  return (
    <Container className="flex mt-40 text-white/60">
      <div className="flex flex-row w-full">
        <FadeIn className="mx-auto text-center mb-12">
          <h2 className="font-display blog-heading-gradient text-[4rem] font-medium leading-[5rem]">
            Changelog
          </h2>
          <p className="font-normal leading-7 mt-6 px-32">
            We are constantly improving our product, fixing bugs and introducing features.
          </p>
          <p>Here you can find the latest updates and changes to Unkey.</p>
        </FadeIn>
      </div>
      <ChangelogGrid changelogs={changelogs} />
      <CTA />
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
  const changelogs = (await getAllMDXData({ contentPath: CHANGELOG_PATH })).sort((a, b) => {
    return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
  });
  return (
    <>
      <Changelog changelogs={changelogs} />
    </>
  );
}
