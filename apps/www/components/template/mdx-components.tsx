import React from "react";
import { Alert } from "../ui/alert/alert";
import { Separator } from "../ui/separator";
import { CodeBlock } from "./codeblock";

export const TemplateComponents = {
  Callout: Alert,
  img: (props: any) => (
    <img {...props} className="object-cover object-center p-0 rounded-3xl" alt="" />
  ),
  th: (props: any) => <th {...props} className="text-base font-semibold text-white" />,
  tr: (props: any) => <tr {...props} className="border-b-[.75px] border-white/10 " />,
  td: (props: any) => <td {...props} className="py-2 text-base font-normal text-white/70" />,

  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60 ellipsis" />,

  ol: (props: any) => <ol {...props} className="pl-4 text-white list-decimal marker:text-white" />,
  ul: (props: any) => (
    <ul {...props} className="pl-4 text-white list-disc sm:pt-4 marker:text-white " />
  ),
  li: (props: any) => <li {...props} className="pl-6 mb-3 mr-2 leading-8 text-white/60" />,
  h1: (props: any) => (
    <h2 {...props} className="text-[32px] font-medium leading-8 blog-heading-gradient " />
  ),
  h2: (props: any) => (
    <h2 {...props} className="text-[32px] font-medium leading-8 blog-heading-gradient " />
  ),
  h3: (props: any) => (
    <h3 {...props} className="text-xl font-medium leading-8 blog-heading-gradient " />
  ),
  h4: (props: any) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient " />
  ),
  p: (props: any) => (
    <p
      {...props}
      className="font-normal leading-8 text-left sm:text-sm md:text-lg text-white/60 "
    />
  ),
  hr: (_props: any) => <Separator orientation="horizontal" />,
  code: (props: any) => (
    <code
      className="px-2 py-1 font-medium text-gray-600 border border-gray-200 rounded-md bg-gray-50 "
      {...props}
    />
  ),
  pre: (props: any) => <CodeBlock {...props} />,
};
