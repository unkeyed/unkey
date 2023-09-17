import { Container } from "@/components/landing/container";
import { PageIntro } from "@/components/landing/page-intro";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ExternalLink, FileQuestion } from "lucide-react";
import Link from "next/link";
import { templates } from "./data";
export const metadata = {
  title: "Templates | Unkey",
  description: "Templates and apps using Unkey",
  openGraph: {
    title: "Templates | Unkey",
    description: "Templates and apps using Unkey",
    url: "https://unkey.dev/templates",
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
    title: "Templates | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/unkey.png",
  },
};

const frameworks = [
  {
    id: "sveltekit",
    name: "SvelteKit",
  },
  {
    id: "nextjs",
    name: "Next.js",
  },
  {
    id: "nuxtjs",
    name: "Nuxt.js",
  },
];

const languages = [
  {
    id: "ts",
    name: "Typescript",
  },
  {
    id: "go",
    name: "Golang",
  },
  {
    id: "rs",
    name: "Rust",
  },
];

export default async function Templates() {
  return (
    <>
      <PageIntro eyebrow="Templates" title="Find your Template">
        <p>Jumpstart your api development with our pre-built solutions.</p>
      </PageIntro>

      <Container className="pt-16 mt-24 border-t">
        <div className="flex space-x-8 ">
          <div className="w-1/4">
            <h2 className="font-semibold ">Filter Templates</h2>
            <Input placeholder="Search ..." className="mt-8" />

            <Accordion type="multiple" className="w-full mt-8">
              <AccordionItem value="item-1">
                <AccordionTrigger>Frameworks</AccordionTrigger>
                <AccordionContent>
                  <div className="flex flex-col space-y-2">
                    {frameworks.map((framework) => (
                      <div
                        className="flex items-center p-3 space-x-2 duration-150 rounded bg-gray-50 hover:bg-gray-100"
                        key={framework.id}
                      >
                        <Checkbox id={framework.id} />
                        <label
                          htmlFor={framework.id}
                          className="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-content"
                        >
                          {framework.name}
                        </label>
                      </div>
                    ))}
                  </div>
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-2">
                <AccordionTrigger>Language</AccordionTrigger>
                <AccordionContent>
                  <div className="flex flex-col space-y-2">
                    {languages.map((language) => (
                      <div
                        className="flex items-center p-3 space-x-2 duration-150 rounded bg-gray-50 hover:bg-gray-100"
                        key={language.id}
                      >
                        <Checkbox id={language.id} />
                        <label
                          htmlFor={language.id}
                          className="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-content"
                        >
                          {language.name}
                        </label>
                      </div>
                    ))}
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>
          <div className="grid w-3/4 grid-cols-1 gap-8 auto-rows-fr lg:grid-cols-3">
            {Object.entries(templates).map(([id, template]) => (
              <Link
                href={`/templates/${id}`}
                key={id}
                className="flex flex-col items-start overflow-hidden duration-200 border border-gray-200 shadow rounded-xl hover:shadow-2xl hover:scale-[1.01]"
              >
                <div className="relative flex justify-center items-center h-full w-full aspect-[16/9] sm:aspect-[2/1] lg:aspect-[3/2]">
                  {template.image ? (
                    <img src={template.image} alt="" className="object-cover w-full " />
                  ) : (
                    <FileQuestion className="w-16 h-16 text-gray-200" />
                  )}
                </div>
                <div className="flex flex-col justify-between h-full px-4 pb-4">
                  <div>
                    <h3 className="mt-3 text-lg font-semibold leading-6 text-gray-900 group-hover:text-gray-600">
                      {template.title}
                    </h3>
                    <p className="mt-5 text-sm leading-6 text-gray-500 line-clamp-3">
                      {template.description}
                    </p>
                  </div>
                  <div className="flex items-center justify-between mt-5">
                    <p className="text-sm leading-6 text-gray-500 ">
                      by {template.authors.join(", ")}
                    </p>
                    <ExternalLink className="w-4 h-4 text-gray-400" />
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </Container>
    </>
  );
}
