import React from "react";
import { Alert } from "../ui/alert/alert";

export const TemplateComponents = {
  img: (props: any) => (
    <img {...props} className="object-cover object-center rounded-3xl p-0" alt="" />
  ),
  Callout: Alert,
  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60" />,

  ol: (props: any) => (
    <ol {...props} className="text-white xl:pl-28 list-decimal marker:text-white" />
  ),
  ul: (props: any) => (
    <ul {...props} className="text-white xxs:pt-4 xl:pl-28 list-disc marker:text-white" />
  ),
  li: (props: any) => <li {...props} className="pl-6 text-white/60" />,
  h1: (_props: any) => null,
  h2: (props: any) => (
    <h2 {...props} className="text-[20px] font-medium leading-8 text-white xl:pl-24" />
  ),
  h3: (props: any) => (
    <h3 {...props} className="text-xl font-medium leading-8 blog-heading-gradient xl:pl-24" />
  ),
  h4: (props: any) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient xl:pl-24" />
  ),
  p: (props: any) => (
    <p {...props} className="text-lg font-normal leading-8 text-white/60 text-left xl:pl-24" />
  ),
  code: (props: any) => (
    <div
      {...props}
      className="inline font-mono text-xs rounded-lg leading-6 font-normal bg-white/10 text-white px-1.5 py-1.5 w-fit text-nowrap overflow-x-auto"
    />
  ),
  pre: (props: any) => (
    <pre
      {...props}
      className="xl:pl-24 bg-transparent my-6 [&>*]:py-6 [&>*]:px-4 [&>*]:block [&>*]:w-full overflow-hidden [&>*]:rounded-xl"
    />
  ),
};
