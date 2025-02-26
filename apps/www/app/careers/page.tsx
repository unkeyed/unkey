import { ChangelogLight } from "@/components/svg/changelog";
import { MeteorLines } from "@/components/ui/meteorLines";

import { allCareers } from "content-collections";
import Link from "next/link";

export default async function Careers() {
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
              <h2 className="blog-heading-gradient text-6xl font-medium mt-12">Open Positions</h2>
              <p className="mt-6 font-normal leading-7 text-balance">
                Unkey is 100% remote. We currently live in Germany, Turkey, Japan and the United
                States.
              </p>
            </div>
          </div>

          <div className="flex flex-row mt-[5.5rem] mb-20 w-full mx-auto container">
            <div className="flex flex-col w-full sm:overflow-hidden gap-16">
              {allCareers
                .filter((c) => c.visible)
                .map((c) => (
                  <Link
                    href={`/careers/${c.slug}`}
                    id={c.slug}
                    key={c.slug}
                    className="w-full flex flex-col gap-2"
                  >
                    <h3 className="font-display text-4xl font-medium blog-heading-gradient ">
                      {c.title}
                    </h3>
                    <p className="text-lg font-normal">{c.description}</p>
                  </Link>
                ))}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

export const metadata = {
  title: "Careers | Unkey",
  description: "Join us.",
  openGraph: {
    title: "Careers | Unkey",
    description: "Join us.",
    url: "https://unkey.com/careers",
    siteName: "unkey.com",
    images: [
      {
        url: "https://unkey.com/og",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Careers | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};
