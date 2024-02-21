import { Container } from "@/components/container";
import { CopyButton } from "@/components/copy-button";
import { CTA } from "@/components/cta";
import { FadeIn } from "@/components/fade-in";
import { Separator } from "@/components/ui/separator";
// import { PageIntro } from "@/components/page-intro";
import { CHANGELOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import Link from "next/link";
import { SideList } from "./side-list";

type Changelog = {
  frontmatter: {
    title: string;
    date: string;
    description: string;
  };
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
      <div className="flex flex-row mt-12">
        <div className="flex flex-col w-96">
          <div className="">
            {/* <div className="absolute top-0 left-0 z-100 w-full h-full bg-color-white"></div> */}
            <SideList logs={changelogs} />
            {/* {changelogs.map((changelog) => (
            <p className="text-sm text-white lg:mt-2 text-left">
              <time
                dateTime={new Date(changelog.frontmatter.date).toDateString()}
              >
                {new Date(changelog.frontmatter.date).toDateString()}
              </time>
            </p>
          ))} */}
          </div>
        </div>
        <div className="flex flex-col ml-20">
          {changelogs.map((changelog) => (
            <FadeIn key={changelog.frontmatter.title}>
              <article>
                <div className="col-span-full sm:flex sm:items-center sm:justify-between sm:gap-x-8 lg:col-span-1 lg:block">
                  <div className="mt-1 flex gap-x-4 sm:mt-0 lg:block">
                    <div className="col-span-full lg:col-span-2 ">
                      <h3 className="font-display text-4xl font-medium blog-heading-gradient">
                        <Link href={`/changelog/${changelog.slug}`}>
                          {changelog.frontmatter.title}
                        </Link>
                      </h3>
                      <p className="mt-10">{changelog.frontmatter.description}</p>
                      <div className="mt-6 mb-6 flex">
                        <Link href={`/changelog/${changelog.slug}`}>
                          <p className="text-white">Read more</p>
                        </Link>
                      </div>
                      <Separator orientation="horizontal" className="mb-6" />
                      <div>
                        <CopyButton
                          value={`https://unkey.dev/changelog/${changelog.slug}`}
                          className="mb-6"
                        >
                          <p className="pl-2">Copy Link</p>
                        </CopyButton>

                        <Separator orientation="horizontal" className="mb-12" />
                      </div>
                    </div>
                  </div>
                </div>
              </article>
            </FadeIn>
          ))}
        </div>
      </div>
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
      {/* <PageIntro eyebrow="" title="Unkey Changelog">
        <p>
          We are constantly improving our product, fixing bugs and introducing features. Here you
          can find the latest updates and changes to Unkey.
        </p>
      </PageIntro> */}

      <Changelog changelogs={changelogs} />
    </>
  );
}
