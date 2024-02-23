import { RainbowDarkButton } from "@/components/button";
import { Container } from "@/components/container";
import { CTA } from "@/components/cta";
import { ChangelogLight, ChangelogLines } from "@/components/svg/changelog";
import { Tags } from "@/lib/mdx-helper";
import { CHANGELOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import { ArrowRight } from "lucide-react";
import { ChangelogGrid } from "./changelog-grid";

type Props = {
  searchParams?: {
    tag?: Tags;
    page?: number;
  };
};

export default async function Changelog(props: Props) {
  const changelogs = (await getAllMDXData({ contentPath: CHANGELOG_PATH })).sort((a, b) => {
    return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
  });
  return (
    <Container className="flex flex-col mt-40 text-white/60 w-full">
      <ChangelogLight className="mx-auto h-1/2" />
      <ChangelogLines className="" />
      <div className="text-center">
        <a href="https://twitter.com/unkeydev">
          <RainbowDarkButton label="Follow us on X" IconRight={ArrowRight} />
        </a>
        <h2 className="blog-heading-gradient text-[4rem] font-medium leading-[5rem] mt-16">
          Changelog
        </h2>
        <p className="font-normal leading-7 mt-6 px-32">
          We are constantly improving our product, fixing bugs and introducing features.
        </p>
        <p>Here you can find the latest updates and changes to Unkey.</p>
      </div>
      <ChangelogGrid changelogs={changelogs} searchParams={props.searchParams} />
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
