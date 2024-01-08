import { allPosts } from "@/.contentlayer/generated";
import { Border } from "@/components/landing/border";
import { Container } from "@/components/landing/container";
import { FadeIn, FadeInStagger } from "@/components/landing/fade-in";
import { GridList, GridListItem } from "@/components/landing/grid-list";
import { PageIntro } from "@/components/landing/page-intro";
import { PageLinks } from "@/components/landing/page-links";
import { SectionIntro } from "@/components/landing/section-intro";
import imageAndreas from "@/images/team/andreas.jpeg";
import imageDom from "@/images/team/dom.jpeg";
import imageJamesPerkins from "@/images/team/james.jpg";
import imageMichael from "@/images/team/michael.png";
import Image from "next/image";
function AboutUnkey() {
  return (
    <div className="py-24 mt-24 rounded-4xl bg-gray-950 sm:mt-32 lg:mt-40 lg:py-32 ">
      <SectionIntro eyebrow="" title="What is Unkey?" invert />
      <Container className="mt-16">
        <GridList>
          <GridListItem title="Globally Founded, Globally Remote" invert>
            We are globally remote and were founded that way too. We believe that the best talent is
            not always in the same place and that we can build a better product by hiring the best
            talent, no matter where they are.
          </GridListItem>
          <GridListItem title="Builders, Innovators" invert>
            We are serial builders who love to innovate. We are always looking for new ways to
            improve our product and our community. If we aren&apos;t working on Unkey, we are
            probably learning about something new.
          </GridListItem>
          <GridListItem title="Open Source" invert>
            Unkey is a fully open source project, we believe that open source leads to better
            products and better communities. We are committed to building a great open source
            community around Unkey and providing the ability to self host for those who want it.
          </GridListItem>
        </GridList>
      </Container>
    </div>
  );
}

const team = [
  {
    title: "Team",
    people: [
      {
        name: "James Perkins",
        role: "Co-Founder / CEO",
        image: { src: imageJamesPerkins },
      },
      {
        name: "Andreas Thomas",
        role: "Co-Founder / CTO",
        image: { src: imageAndreas },
      },
      {
        name: "Michael Silva",
        role: "Developer",
        image: { src: imageMichael },
      },
      {
        name: "Dom Eccleston",
        role: "Developer",
        image: { src: imageDom },
      },
    ],
  },
];

function Team() {
  return (
    <Container className="mt-24 sm:mt-32 lg:mt-40">
      <div className="space-y-24">
        {team.map((group) => (
          <FadeInStagger key={group.title}>
            <Border as={FadeIn} />
            <div className="grid grid-cols-1 gap-6 pt-12 sm:pt-16 lg:grid-cols-4 xl:gap-8">
              <FadeIn>
                <h2 className="font-sans text-2xl font-semibold font-display text-gray-950">
                  {group.title}
                </h2>
              </FadeIn>
              <div className="lg:col-span-4">
                <ul className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4 xl:gap-8">
                  {group.people.map((person) => (
                    <li key={person.name}>
                      <FadeIn>
                        <div className="relative overflow-hidden group rounded-3xl bg-gray-100">
                          <Image
                            alt=""
                            {...person.image}
                            className="object-cover w-full transition duration-500 h-96 grayscale motion-safe:group-hover:scale-105"
                          />
                          <div className="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-black to-black/0 to-40% p-6">
                            <p className="font-semibold tracking-wide text-white font-display text-base/6">
                              {person.name}
                            </p>
                            <p className="mt-2 text-sm text-white">{person.role}</p>
                          </div>
                        </div>
                      </FadeIn>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </FadeInStagger>
        ))}
      </div>
    </Container>
  );
}

export const metadata = {
  title: "About Us | Unkey",
  description:
    "Learn about the Unkey team and our mission to build a better API authentication solution.",
  openGraph: {
    title: "About Us | Unkey",
    description:
      "Learn about the Unkey team and our mission to build a better API authentication solution.",
    url: "https://unkey.dev/pricing",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "About Us | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

export default async function About() {
  const blogArticles = allPosts
    .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
    .slice(0, 2);
  return (
    <>
      <PageIntro eyebrow="" title="About us">
        <p>
          Unkey is an <span className="font-semibold">open source</span> API authentication and
          authorization platform for scaling user facing APIs. Create, verify, and manage low
          latency API keys in seconds.
        </p>
        <div className="max-w-2xl mt-10 space-y-6 text-base">
          <p>
            Unkey was started by James Perkins and Andreas Thomas in 2023. We were frustrated by the
            lack of a simple, easy to use API authentication solution that was also fast and
            scalable. We decided to build our own solution and Unkey was born.
          </p>
        </div>
      </PageIntro>

      <AboutUnkey />

      <Team />

      <PageLinks
        className="mt-24 sm:mt-32 lg:mt-40"
        title="From the blog"
        intro="We write about latest trends in development, and changes to Unkey"
        pages={blogArticles}
      />
    </>
  );
}
