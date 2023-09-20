import { Container } from "@/components/landing/container";
import { FadeIn, FadeInStagger } from "@/components/landing/fade-in";
import { List, ListItem } from "@/components/landing/list";
import { SectionIntro } from "@/components/landing/section-intro";
import { StatList, StatListItem } from "@/components/landing/stat-list";
import { StylizedImage } from "@/components/landing/stylized-image";
import { Button } from "@/components/ui/button";
import laptopImage from "@/images/computer-user.jpg";
import { db, schema, sql } from "@/lib/db";
import { getTotalVerifications } from "@/lib/tinybird";
import { Github } from "lucide-react";
import Link from "next/link";
export const revalidate = 60;

const [workspaces, apis, keys, totalVerifications] = await Promise.all([
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.workspaces)
    .then((res) => res.at(0)?.count ?? 0),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.apis)
    .then((res) => res.at(0)?.count ?? 0),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .then((res) => res.at(0)?.count ?? 0),
  getTotalVerifications({}).then((res) => res.data.at(0)?.total ?? 0),
]);

function NumbersServed() {
  return (
    <div className="mt-24 rounded-4xl sm:mt-32  lg:mt-32">
      <Container className="">
        <FadeIn className="flex items-center gap-x-8">
          <h2 className="mb-8 text-2xl font-semibold tracking-wider text-center text-black font-display sm:text-left">
            We serve
          </h2>
          <div className="flex-auto h-px" />
        </FadeIn>
        <FadeInStagger faster>
          <StatList>
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(workspaces)}
              label="Workspaces"
            />
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(apis)}
              label="APIs"
            />
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(keys)}
              label="Keys"
            />
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(totalVerifications)}
              label="Verifications"
            />
          </StatList>
        </FadeInStagger>
      </Container>
    </div>
  );
}

function Features() {
  return (
    <>
      <SectionIntro
        eyebrow=""
        title="Features for any use case"
        className="mt-24 sm:mt-32 lg:mt-32"
      >
        <p>
          Whether you are working on your latest side project or building the next big thing, Unkey
          has you covered.
        </p>
      </SectionIntro>
      <Container className="mt-16 overflow-x-hidden	">
        <div className="lg:flex lg:items-center lg:justify-end">
          <div className="flex justify-center lg:w-1/2 lg:justify-end lg:pr-12">
            <FadeIn className="w-[33.75rem] flex-none lg:w-[45rem]">
              <StylizedImage
                src={laptopImage}
                sizes="(min-width: 1024px) 41rem, 31rem"
                className="justify-center lg:justify-end w-full"
              />
            </FadeIn>
          </div>
          <List className="mt-16 lg:mt-0 lg:w-1/2 lg:min-w-[33rem] lg:pl-4">
            <ListItem title="Keys where your users are">
              Your users are everywhere and so is Unkey. Unkey stores keys globally, making each
              request as fast possible regardless of your location.
            </ListItem>
            <ListItem title="Per-key rate limiting">
              We understand that each user is different, so Unkey gives you the ability to decide
              the rate limits as you issue each key. Giving you complete control while protecting
              your application.
            </ListItem>
            <ListItem title="Temporary Keys">
              Want to add a free trial to your API? Unkey allows you to issue temporary keys, once
              the key expires we delete it for you.
            </ListItem>
            <ListItem title="Limited Keys">
              Want to limit the number of requests a user can make? Unkey allows you to issue
              limited keys, once the key reaches the limit we delete it for you.
            </ListItem>
          </List>
        </div>
      </Container>
    </>
  );
}

export const metadata = {
  title: "Unkey",
  description: "Accelerate your API Development",
  openGraph: {
    title: "Unkey",
    description: "Accelerate your API Development",
    url: "https://unkey.dev/",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/unkey.png",
  },
};

export default async function Home() {
  return (
    <>
      <Container className="md:mb-8 mt-24 sm:mt-32 md:mt-56 lg:px-0">
        <FadeIn className="flex flex-col md:flex-row md:justify-stretch md:spacing-x-4">
          <div className="w-full">
            <h1 className="font-display text-5xl font-medium tracking-tight text-gray-950 [text-wrap:balance] sm:text-7xl font-sans">
              API authentication made easy
            </h1>
            <p className={"mt-6 text-xl text-gray-600 [text-wrap:balance]"}>
              Seriously fast and easy to use. Unkey is an{" "}
              <span className="font-semibold">open source</span> API management platform that helps
              developers secure, manage, and scale their APIs.
            </p>
            <div className="flex flex-col md:flex-row mt-4 md:space-x-8 space-y-4 md:space-y-0">
              <Button size="xl" className="rounded-full text-sm font-semibold" asChild>
                <Link className="flex-none" href="/app">
                  Start for free
                </Link>
              </Button>
              <Button
                size="xl"
                className="rounded-full text-sm font-semibold"
                variant="secondary"
                asChild
              >
                <a
                  className="flex-none"
                  href="https://github.com/unkeyed/unkey"
                  target="_blank"
                  rel="noreferrer"
                >
                  <Github className="mr-2 h-4 w-4" /> Star us on GitHub
                </a>
              </Button>
            </div>
          </div>
          <div className="hidden lg:block lg:ml-12">
            <iframe
              width="560"
              height="315"
              src="https://www.youtube-nocookie.com/embed/-gvpo4SWgG8?si=8n5wIAdTZA7PCgZb&amp;controls=0"
              title="Unkey in Five minutes - YouTube"
              frameborder="0"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowfullscreen
            />
          </div>
          <div className="mt-4 aspect-w-16 aspect-h-9 lg:hidden">
            <iframe
              src="https://www.youtube-nocookie.com/embed/-gvpo4SWgG8?si=8n5wIAdTZA7PCgZb"
              frameborder="0"
              title="Unkey in Five minutes - YouTube"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowfullscreen
            />
          </div>
        </FadeIn>
      </Container>
      <NumbersServed />

      <Features />
    </>
  );
}
