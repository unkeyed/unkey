import { Container } from "@/components/container";
import { CTA } from "@/components/cta";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { cn } from "@/lib/utils";

export const metadata = {
  title: "Playground | Unkey",
  description: "Try unkey without signing up.",
  openGraph: {
    title: "About | Unkey",
    description: "Learn more about Unkey and how we operate.",
    url: "https://unkey.com/playground",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.com/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Playground | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

export default async function Page() {
  return (
    <Container className="text-lg">
      <div className="mt-32">
        <h2 className="text-2xl">Unkey Playground</h2>
      </div>
      <div className="flex w-full mt-8 h-full">
        <div className="flex flex-col w-full h-full">
          <Accordion type="single" collapsible className="w-full">
            <AccordionItem value="item-1" className="">
              <AccordionTrigger className="px-4 justify-between bg-slate-900 rounded-2xl">
                1. Create Key
              </AccordionTrigger>
              <AccordionContent className="p-4 bg-slate-950 rounded-2xl px-6">
                <p className="text-center leading-7">There are two methods to create a key.</p>
                <ol className="list-decimal pl-4 text-lg leading-7 pt-2">
                  <li>
                    You can create a key by clicking on the "Create Key" button on the top right
                    corner of your api page in our dashboard.
                  </li>
                  <li>You can also create a key using Unkey's API.</li>
                </ol>
                {/* <button className={cn("mt-4", "button", "button-primary")}>Create Key</button> */}
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="item-2">
              <AccordionTrigger className="px-4 justify-between">2. Verify Key</AccordionTrigger>
              <AccordionContent className="p-4">
                Yes. It comes with default styles that matches the other components&apos; aesthetic.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="item-3">
              <AccordionTrigger className="px-4 justify-between">3. Update Key</AccordionTrigger>
              <AccordionContent className="p-4">
                Yes. It&apos;s animated by default, but you can disable it if you prefer.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="item-3">
              <AccordionTrigger className="px-4 justify-between">4. Verify key</AccordionTrigger>
              <AccordionContent className="p-4">
                Yes. It&apos;s animated by default, but you can disable it if you prefer.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="item-3">
              <AccordionTrigger className="px-4 justify-between">5. Get Analytics</AccordionTrigger>
              <AccordionContent className="p-4">
                Yes. It&apos;s animated by default, but you can disable it if you prefer.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="item-3">
              <AccordionTrigger className="px-4 justify-between">6. Revoke Key</AccordionTrigger>
              <AccordionContent className="p-4">
                Yes. It&apos;s animated by default, but you can disable it if you prefer.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="item-3">
              <AccordionTrigger className="px-4 justify-between">7. Verify Key</AccordionTrigger>
              <AccordionContent className="p-4">
                Yes. It&apos;s animated by default, but you can disable it if you prefer.
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
        <div className="flex w-full h-full p-6">
          <input
            className="w-full h-10 bg-slate-800 text-white rounded-lg  p-4 ring-2 ring-slate-900 text-pretty justify-end"
            placeholder="Playground/unkey > "
          />
        </div>
      </div>
      <CTA />
    </Container>
  );
}
