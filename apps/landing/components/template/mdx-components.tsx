import React from "react";
import { BlogQuote } from "../blog-quote";
import { Alert } from "../ui/alert/alert";

export const TemplateComponents = {
  img: (props: any) => (
    <img {...props} className="object-cover object-center rounded-3xl p-0" alt="" />
  ),
  Callout: Alert,
  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60" />,
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  ol: (props: any) => (
    <ol {...props} className="text-white lg:pl-28 list-decimal marker:text-white" />
  ),
  ul: (props: any) => <ul {...props} className="text-white lg:pl-28 list-disc marker:text-white" />,
  li: (props: any) => <li {...props} className="pl-6 text-white/60" />,
  h1: (props: any) => (
    <h1 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient pl-24" />
  ),
  h2: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient pl-24" />
  ),
  h3: (props: any) => (
    <h3 {...props} className="text-xl font-medium leading-8 blog-heading-gradient pl-24" />
  ),
  h4: (props: any) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient pl-24" />
  ),
  p: (props: any) => (
    <p {...props} className="text-lg font-normal leading-8 text-white/60 text-left pl-24" />
  ),
  code: (props: any) => (
    <div
      {...props}
      className="font-mono rounded-md bg-white/10 text-white/60 px-3 pt-0.5 pb-1 mx-1 w-fit"
    />
  ),
  pre: (props: any) => <div {...props} className="font-mono rounded-md bg-transparent ml-24" />,
};
