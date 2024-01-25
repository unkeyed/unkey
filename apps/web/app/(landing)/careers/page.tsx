import { Border } from "@/components/landing/border";
import { Button } from "@/components/landing/button";
import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { JOBS_PATH, getAllMDXData } from "@/lib/mdx-helper";
import Link from "next/link";

export const metadata = {
  title: "Jobs | Unkey",
  description: "Join the team at Unkey and help us build the future of API auth.",
  openGraph: {
    title: "Jobs | Unkey",
    description: "Join the team at Unkey and help us build the future of API auth.",
    url: "https://unkey.dev/careers",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/og/career",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Jobs | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/unkey.png",
  },
};

export default async function JobsPage() {
  const jobs = (await getAllMDXData({ contentPath: JOBS_PATH })).filter((job) => {
    return job.frontmatter.visible !== false;
  });

  return (
    <>
      <PageIntro eyebrow="" title="Join Us.">
        <p>We're solving API auth for the next generation of developers.</p>
      </PageIntro>

      <Container className="mt-40">
        <FadeIn>
          <h2 className="font-display text-2xl font-semibold text-gray-950">Open Positions</h2>
        </FadeIn>
        <div className="mt-10 space-y-20 sm:space-y-24 lg:space-y-32">
          {jobs.length === 0 && (
            <FadeIn>
              <p className="text-lg text-neutral-950">
                We don't have any open positions right now. Check back later!
              </p>
            </FadeIn>
          )}
          {jobs.map((job) => (
            <FadeIn key={job.frontmatter.title}>
              <article>
                <Border className="grid grid-cols-3 gap-x-8 gap-y-8 pt-16">
                  <div className="col-span-full sm:flex sm:items-center sm:justify-between sm:gap-x-8 lg:col-span-1 lg:block">
                    <div className="sm:flex sm:items-center sm:gap-x-6 lg:block">
                      <h3 className=" text-sm font-semibold text-neutral-950">
                        {job.frontmatter.level}
                      </h3>
                    </div>
                    <div className="mt-1 flex gap-x-4 sm:mt-0 lg:block">
                      <p className="text-sm tracking-tight text-neutral-950 after:ml-4 after:font-semibold after:text-neutral-300 lg:mt-2 lg:after:hidden">
                        {job.frontmatter.description}
                      </p>
                    </div>
                  </div>
                  <div className="col-span-full lg:col-span-2 lg:max-w-2xl">
                    <p className="font-display text-4xl font-medium text-gray-950">
                      <Link href={job.slug}>{job.frontmatter.title}</Link>
                    </p>

                    <div className="mt-8 flex">
                      <Button
                        href={job.slug}
                        aria-label={`Read the description for ${job.frontmatter.title}`}
                      >
                        Read more
                      </Button>
                    </div>
                  </div>
                </Border>
              </article>
            </FadeIn>
          ))}
        </div>
      </Container>
    </>
  );
}
