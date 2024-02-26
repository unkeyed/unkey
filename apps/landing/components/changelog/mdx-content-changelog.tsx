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
  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60 ellipsis" />,

  ol: (props: any) => <ol {...props} className="text-white list-decimal marker:text-white " />,
  ul: (props: any) => (
    <ul {...props} className="text-white xxs:pt-4 list-disc marker:text-white " />
  ),
  li: (props: any) => <li {...props} className="pl-6 text-white/60" />,
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
    <div
      {...props}
      className="inline font-mono text-xs rounded-lg leading-6 xxs:text-xs md:text-base font-normal bg-white/10 text-white px-1.5 py-1.5 w-full text-nowrap overflow-x-auto"
    />
  ),
  pre: (props: any) => (
    <pre
      {...props}
      className="bg-transparent my-6 [&>*]:py-6 [&>*]:px-4 [&>*]:block w-full [&>*]:rounded-xl sm:m-0 sm:p-0"
    />
  ),
};

export function MdxContentChangelog({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
