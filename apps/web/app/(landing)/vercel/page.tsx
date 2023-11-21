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
import { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Vercel Integration",
  description: "Zero Config API Authentication",
};

export default function Vercel() {
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
              allowFullScreen
            />
          </div>
          <div className="mt-4 aspect-w-16 aspect-h-9 lg:hidden">
            <iframe
              src="https://www.youtube-nocookie.com/embed/fDKkicMZiCc?si=ksblX5j1-OUvNLpf&amp;controls=0"
              title="Unkey in Five minutes - YouTube"
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
              allowFullScreen
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
