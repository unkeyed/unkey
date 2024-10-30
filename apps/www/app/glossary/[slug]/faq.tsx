import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

export function FAQ(props: {
  epigraph?: string;
  title: string;
  description: string;
  items: Array<{ question: string; answer: string }>;
}) {
  return (
    <>
      <div className="text-center space-y-4 pb-6 mx-auto">
        <h3 className="not-prose blog-heading-gradient text-left font-medium tracking-tight text-4xl">
          {props.title}
        </h3>
        <p className="font-medium leading-7 not-prose text-white/70 lg:text-xl text-sm md:text-base py-6 max-w-sm md:max-w-md lg:max-w-xl xl:max-w-4xl text-left">
          {props.description}
        </p>
      </div>
      <div className="mx-auto md:max-w-[800px]">
        <Accordion type="single" collapsible className="w-full">
          {props.items.map((item) => (
            <AccordionItem value={item.question} key={item.question}>
              <AccordionTrigger className="justify-between space-x-2">
                {item.question}
              </AccordionTrigger>
              <AccordionContent>{item.answer}</AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </>
  );
}
