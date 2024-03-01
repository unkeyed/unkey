"use client";

import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import { BlogImage } from "../../app/blog/blog-image";
import { Alert } from "../ui/alert/alert";
import { Separator } from "../ui/separator";

type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: (props: any) => <BlogImage size="lg" imageUrl={props} />,
  Callout: Alert,
  img: (props: any) => (
    <img {...props} className="object-cover object-center rounded-3xl p-0" alt="" />
  ),
  th: (props: any) => (
    <th {...props} className="text-white font-semibold text-base text-left pb-4" />
  ),
  tr: (props: any) => <tr {...props} className="border-b-[.75px] border-white/10" />,
  td: (props: any) => <td {...props} className="text-white/70 text-base font-normal py-4" />,

  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60 ellipsis" />,

  ol: (props: any) => (
    <ol {...props} className="text-white list-decimal marker:text-white xxs:mx-4" />
  ),
  ul: (props: any) => (
    <ul {...props} className="text-white xxs:pt-4 list-disc marker:text-white md:pt-0 xxs:mx-4 " />
  ),
  li: (props: any) => <li {...props} className="pl-6 text-white/60 xxs:mb-4" />,
  h1: (_props: any) => null,
  h2: (props: any) => <h2 {...props} className="text-[32px] font-medium leading-8 text-white " />,
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
      className="inline font-mono text-xs rounded-lg leading-6 xxs:text-xs md:text-base font-normal bg-white/10 text-white px-1.5 py-1.5 w-full text-nowrap overflow-x-auto"
    />
  ),
  pre: (props: any) => (
    <pre
      {...props}
      className="bg-transparent [&>*]my-6 [&>*]:py-6 [&>*]:px-4 [&>*]:block w-full [&>*]:rounded-xl m-0 p-0"
    />
  ),
};

export function MdxContentChangelog({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
