import { Text } from "@/components/dashboard/text";
import { Container } from "@/components/landing/container";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { ExternalLink } from "lucide-react";
import Link from "next/link";
export default function Example() {
  return (
    <Container>
      <div className="relative flex flex-col items-start mt-16 space-y-8 lg:flex-row lg:mt-32 lg:space-y-0">
        <div className="self-start w-full px-4 mx-auto lg:sticky top-32 h-max lg:w-2/5 sm:px-6 lg:px-8 ">
          <div className="pb-10 mt-4">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-6xl">
              Zero Config API Authentication
            </h2>
            <p className="mt-2 text-gray-500">
              {" "}
              Integrate Unkey in your Vercel workflow, and we'll automatically inject your
              environment variables. No more manual configuration or copy and pasting.
            </p>
          </div>
          <div className="flex items-center justify-between gap-4">
            <Link target="_blank" href="https://vercel.com/integrations/unkey" className="w-1/2">
              <Button size="lg" className="w-full whitespace-nowrap">
                Connect your Project
              </Button>
            </Link>
            <Link
              target="_blank"
              href="https://unkey.dev/docs/integrations/vercel"
              className="w-1/2"
            >
              <Button variant="secondary" size="lg" className="w-full whitespace-nowrap">
                Read more
                <ExternalLink className="inline-block w-3 h-3 ml-1" />
              </Button>
            </Link>
          </div>
        </div>

        <div className="w-full border-gray-100 lg:border-l lg:pl-8 lg:w-3/5">
          <div className="hidden lg:block lg:ml-12">
            <iframe
              width="560"
              height="315"
              src="https://www.youtube-nocookie.com/embed/fDKkicMZiCc?si=ksblX5j1-OUvNLpf&amp;controls=0"
              title="Unkey in Five minutes - YouTube"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowFullscreen
            />
          </div>
          <div className="mt-4 aspect-w-16 aspect-h-9 lg:hidden">
            <iframe
              src="https://www.youtube-nocookie.com/embed/fDKkicMZiCc?si=ksblX5j1-OUvNLpf&amp;controls=0"
              title="Unkey in Five minutes - YouTube"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowFullscreen
            />
          </div>
          <div className="flex flex-col max-w-2xl mx-auto mt-16">
            <h2 className="text-2xl">FAQs</h2>
            <Accordion type="multiple" className="w-full">
              <AccordionItem value="what-is-unkey">
                <AccordionTrigger dir="">What is Unkey?</AccordionTrigger>
                <AccordionContent>
                  <Text>
                    Unkey is an open source API management platform that helps developers secure,
                    manage, and scale their APIs. Unkey has built-in features that can make it
                    easier than ever to provide an API to your end users
                  </Text>
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="unkey-integrate">
                <AccordionTrigger dir="">How does Unkey integrate with Vercel?</AccordionTrigger>
                <AccordionContent>
                  Once Unkey is connected with Vercel, you can select which workspace should be
                  connected to your Vercel deployment. We will then insert all of your environment
                  variables required to use Unkey.
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="unkey-free">
                <AccordionTrigger dir="">Is Unkey Free?</AccordionTrigger>
                <AccordionContent>
                  Unkey has a generous free tier and is open source. We also offer a pro tier for
                  businesses and high usage projects. Check out our{" "}
                  <Link className="underline" href="/pricing">
                    pricing page
                  </Link>{" "}
                  for more details.
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>
        </div>
      </div>
    </Container>
  );
}

const _Rings: React.FC = (): JSX.Element => {
  return (
    <div className="absolute left-1/2  h-2/3 scale-150  stroke-zinc-700/70 [mask-image:linear-gradient(to_top,white_20%,transparent_75%)] -translate-x-1/2">
      {/* Outer ring */}

      <svg
        viewBox="0 0 1026 1026"
        fill="none"
        aria-hidden="true"
        className="inset-0 w-full h-full animate-spin-forward-slow"
      >
        <path
          d="M1025 513c0 282.77-229.23 512-512 512S1 795.77 1 513 230.23 1 513 1s512 229.23 512 512Z"
          stroke="#d4d4d8"
          strokeOpacity="0.7"
        />
        <path
          d="M513 1025C230.23 1025 1 795.77 1 513"
          stroke="url(#gradient-1)"
          strokeLinecap="round"
        />
        <defs>
          <linearGradient
            id="gradient-1"
            x1="1"
            y1="513"
            x2="1"
            y2="1025"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#0000aa" />
            <stop offset={1} stopColor="#121212" stopOpacity={0} />
          </linearGradient>
        </defs>
      </svg>
      {/* Inner ring */}
      <svg
        viewBox="0 0 1026 1026"
        fill="none"
        aria-hidden="true"
        className="absolute inset-0 w-full h-full animate-spin-reverse-slower"
      >
        <path
          d="M913 513c0 220.914-179.086 400-400 400S113 733.914 113 513s179.086-400 400-400 400 179.086 400 400Z"
          stroke="#d4d4d8"
          strokeOpacity="0.7"
        />
        <path
          d="M913 513c0 220.914-179.086 400-400 400"
          stroke="url(#gradient-2)"
          strokeLinecap="round"
        />
        <defs>
          <linearGradient
            id="gradient-2"
            x1="913"
            y1="513"
            x2="913"
            y2="913"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#0000aa" />
            <stop offset={1} stopColor="#121212" stopOpacity={0} />
          </linearGradient>
        </defs>
      </svg>
    </div>
  );
};
