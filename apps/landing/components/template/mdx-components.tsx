import React from "react";
import { Alert } from "../ui/alert/alert";
import { Separator } from "../ui/separator";
import { CodeBlock } from "./codeblock";

export const TemplateComponents = {
  Callout: Alert,
  img: (props: any) => (
    <img {...props} className="object-cover object-center rounded-3xl p-0" alt="" />
  ),
  th: (props: any) => <th {...props} className="text-white font-semibold text-base" />,
  tr: (props: any) => <tr {...props} className="border-b-[.75px] border-white/10 " />,
  td: (props: any) => <td {...props} className="text-white/70 text-base font-normal py-2" />,

  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60 ellipsis" />,

  ol: (props: any) => <ol {...props} className="text-white list-decimal marker:text-white pl-4" />,
  ul: (props: any) => (
    <ul {...props} className="text-white sm:pt-4 list-disc marker:text-white pl-4 " />
  ),
  li: (props: any) => <li {...props} className="pl-6 text-white/60 mb-3 leading-8 mr-2" />,
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
      className="sm:text-sm md:text-lg font-normal leading-8 text-white/60 text-left "
    />
  ),
  hr: (_props: any) => <Separator orientation="horizontal" />,
  code: (props: any) => (
    <code
      {...props}
      className="inline font-mono text-xs rounded-lg leading-6 sm:text-xs md:text-base font-normal bg-white/10 text-white px-1.5 py-1 w-full text-nowrap overflow-x-auto"
    />
  ),
  pre: (props: any) => <CodeBlock {...props} />,
};
