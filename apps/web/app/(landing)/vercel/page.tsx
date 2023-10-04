import Link from "next/link";
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
  } from "@/components/ui/accordion";
export default function Example() {
  return (
    <div>
      <div className="px-6 py-24 mx-auto max-w-7xl sm:py-32 lg:flex lg:items-center lg:gap-x-10 lg:px-8 lg:py-40">
        <div className="max-w-2xl mx-auto lg:mx-0 lg:flex-auto">
          <h1 className="max-w-lg mt-10 text-7xl font-bold tracking-tight text-gray-900 sm:text-6xl">
            Zero config API Authentication
          </h1>
          <p className="mt-6 text-lg leading-8 text-gray-600">
            Integrate your Unkey project with Vercel copy blah blah. 
          </p>
          <div className="flex items-center mt-10 gap-x-6">
            <Link
              href="/auth/sign-up"
              className="rounded-md border border-primary bg-primary px-3.5 py-1.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-secondary hover:text-secondary-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
            >
              Integrate for free
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
                            something
                            </AccordionContent>
                    </AccordionItem>
                    <AccordionItem value="unkey-integrate">
                          <AccordionTrigger dir="">How does Unkey integrate with Vercel?</AccordionTrigger>
                          <AccordionContent>
                            something
                            </AccordionContent>
                    </AccordionItem>
                    <AccordionItem value="unkey-free">
                          <AccordionTrigger dir="">Is Unkey Free?</AccordionTrigger>
                          <AccordionContent>
                            something
                            </AccordionContent>
                    </AccordionItem>

        </Accordion>
        </div>
    </div>
  );
}
