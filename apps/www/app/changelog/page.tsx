import { RainbowDarkButton } from "@/components/button";
import { CTA } from "@/components/cta";
import { ChangelogLight } from "@/components/svg/changelog";
import { MeteorLines } from "@/components/ui/meteorLines";

import { allChangelogs } from "@/.contentlayer/generated";
import { ChangelogGridItem } from "@/components/changelog/changelog-grid-item";
import { SideList } from "@/components/changelog/side-list";
import { ArrowRight } from "lucide-react";
type Props = {
  searchParams?: {
    tag?: string[];
    page?: number;
  };
};

export default async function Changelogs(_props: Props) {
  const changelogs = allChangelogs.sort((a, b) => {
    return new Date(b.date).getTime() - new Date(a.date).getTime();
  });

  return (
    <>
      <div className="container mt-48 text-white/60">
        <div>
          <div className="relative -z-100 max-w-[1000px] mx-auto">
            <ChangelogLight className="w-full -top-[20rem]" />
          </div>
          <div className="w-full">
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
          <div className="flex flex-row text-center">
            <div className="mx-auto flex-flex-col ">
              <a href="https://twitter.com/unkeydev" target="_blank" rel="noreferrer">
                <RainbowDarkButton label="Follow us on X" IconRight={ArrowRight} />
              </a>
              <h2 className="blog-heading-gradient text-6xl font-medium mt-12">Changelog</h2>
              <p className="mt-6 font-normal leading-7 text-balance">
                We are constantly improving our product, fixing bugs and introducing features.{" "}
                <br className="hidden lg:inline" />
                Here you can find the latest updates and changes to Unkey.
              </p>
            </div>
          </div>

          <div className="flex flex-row mt-[5.5rem] gap-20 mb-20 w-full mx-auto">
            <div className="relative hidden w-72 lg:block">
              <div className="top-20 sticky">
                <SideList logs={changelogs} className="" />
              </div>
            </div>
            <div className="flex flex-col w-full sm:overflow-hidden">
              {changelogs?.map((changelog) => (
                <ChangelogGridItem key={changelog.title} changelog={changelog} />
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
    url: "https://unkey.com/changelog",
    siteName: "unkey.com",
    images: [
      {
        url: "https://unkey.com/og/changelog",
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
