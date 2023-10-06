import Link from "next/link";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Text } from "@/components/dashboard/text";
export default function Example() {
  return (
    <div>
      <div className="px-6 py-24 mx-auto max-w-7xl sm:py-32 lg:flex lg:items-center lg:gap-x-10 lg:px-8 lg:py-40">
        <div className="max-w-2xl mx-auto lg:mx-0 lg:flex-auto">
          <h1 className="max-w-lg mt-10 text-7xl font-bold tracking-tight text-gray-900 sm:text-6xl">
            Zero config API Authentication
          </h1>
          <p className="mt-6 text-lg leading-8 text-gray-600">
            Integrate Unkey in your Vercel workflow, and we'll automatically inject your environment variables. No more manual configuration or copy and pasting.
          </p>
          <div className="flex items-center mt-10 gap-x-6">
            <Link
              href="/todo"
              className="rounded-md border border-primary bg-primary px-3.5 py-1.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-secondary hover:text-secondary-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
            >
              Integrate for free
            </Link>
            <Link
              href="/todo"
              className="rounded-md border border-primary bg-primary px-3.5 py-1.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-secondary hover:text-secondary-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
            >
              Read our integration docs
            </Link>
          </div>
        </div>
        <div className="mt-16 sm:mt-24 lg:mt-0 lg:flex-shrink-0 lg:flex-grow">
          <img
            alt="Vercel and Unkey"
            className="mx-auto w-[36rem] max-w-full drop-shadow-xl rounded-lg"
            src="/unkey-vercel.png"
          />
        </div>

      </div>
      <div className="flex flex-col mx-auto max-w-2xl">
        <h2 className="text-2xl">FAQs</h2>
        <Accordion type="multiple" className="w-full">
          <AccordionItem value="what-is-unkey">
            <AccordionTrigger dir="">What is Unkey?</AccordionTrigger>
            <AccordionContent>
              <Text>Unkey is an open source API management platform that helps developers secure, manage, and scale their APIs. Unkey has built-in features that can make it easier than ever to provide an API to your end users</Text>
            </AccordionContent>
          </AccordionItem>
          <AccordionItem value="unkey-integrate">
            <AccordionTrigger dir="">How does Unkey integrate with Vercel?</AccordionTrigger>
            <AccordionContent>
            Once Unkey is connected with Vercel, you can select which workspace should be connected to your Vercel deployment. We will then insert all of your environment variables required to use Unkey.
            </AccordionContent>
          </AccordionItem>
          <AccordionItem value="unkey-free">
            <AccordionTrigger dir="">Is Unkey Free?</AccordionTrigger>
            <AccordionContent>
              Unkey has a generous free tier and is open source. We also offer a pro tier for businesses and high usage projects. Check out our <Link className="underline" href="/pricing">pricing page</Link> for more details.
            </AccordionContent>
          </AccordionItem>

        </Accordion>
      </div>
    </div>
  );
}
