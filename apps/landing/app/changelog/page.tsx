import { RainbowDarkButton } from "@/components/button";
import { CTA } from "@/components/cta";
import { ChangelogLight } from "@/components/svg/changelog";
import { MeteorLines } from "@/components/ui/meteorLines";
import { Tags } from "@/lib/mdx-helper";
import { CHANGELOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import { ArrowRight } from "lucide-react";
import { ChangelogGridItem } from "./changelog-grid-item";
import { SideList } from "./side-list";

type Props = {
  searchParams?: {
    tag?: Tags;
    page?: number;
  };
};

export default async function Changelog(_props: Props) {
  const changelogs = (await getAllMDXData({ contentPath: CHANGELOG_PATH })).sort((a, b) => {
    return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
  });

  return (
    <>
      <div className="flex flex-col mt-32 text-white/60 mx-auto">
        <div>
          <div className="relative -z-100 max-w-[1000px] mx-auto">
            <ChangelogLight className="w-full" />
          </div>
          <div className="w-full overflow-hidden">
            <MeteorLines number={1} xPos={60} direction="left" speed={10} delay={0} />
            <MeteorLines number={1} xPos={60} direction="left" speed={10} delay={5} />

            <MeteorLines number={1} xPos={200} direction="left" speed={10} delay={4} />
            <MeteorLines number={1} xPos={200} direction="left" speed={10} delay={8} />

            <MeteorLines
              className="hidden sm:block"
              number={1}
              xPos={350}
              direction="left"
              speed={10}
              delay={2}
            />
            <MeteorLines
              className="hidden sm:block"
              number={1}
              xPos={350}
              direction="left"
              speed={10}
              delay={8}
            />
            <MeteorLines number={1} xPos={60} direction="right" speed={10} delay={0} />
            <MeteorLines number={1} xPos={60} direction="right" speed={10} delay={5} />

            <MeteorLines number={1} xPos={200} direction="right" speed={10} delay={4} />
            <MeteorLines number={1} xPos={200} direction="right" speed={10} delay={8} />

            <MeteorLines
              className="hidden sm:block"
              number={1}
              xPos={350}
              direction="right"
              speed={10}
              delay={2}
            />
            <MeteorLines
              className="hidden sm:block"
              number={1}
              xPos={350}
              direction="right"
              speed={10}
              delay={8}
            />
          </div>
        </div>
        <div>
          <div className="text-center flex flex-row mx-auto ">
            <div className="flex-flex-col mx-auto px-4">
              <a href="https://twitter.com/unkeydev" target="_blank" rel="noreferrer">
                <RainbowDarkButton label="Follow us on X" IconRight={ArrowRight} />
              </a>
              <h2 className="blog-heading-gradient text-[4rem] font-medium leading-[5rem] mt-16">
                Changelog
              </h2>
              <p className="font-normal leading-7 mt-6 ">
                We are constantly improving our product, fixing bugs and introducing features.
              </p>
              <p>Here you can find the latest updates and changes to Unkey.</p>
            </div>
          </div>

          <div className="flex flex-row mt-20 mb-20 max-w-[1400px] w-full mx-auto">
            <div className="relative w-80 mx-12 hidden xl:block">
              <div className="top-12 left-0  sticky ">
                <SideList logs={changelogs} className="mt-0 pt-0   changlog-gradient" />
              </div>
            </div>
            <div className="flex flex-col w-full sm:overflow-hidden">
              {changelogs.map((changelog) => (
                <ChangelogGridItem key={changelog.slug} changelog={changelog} />
              ))}
            </div>
          </div>
        </div>
      </div>
      <CTA />
    </>
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
