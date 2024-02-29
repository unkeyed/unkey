"use client";
import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import { BlogCodeBlock } from "../app/blog/blog-code-block";
import { BlogImage } from "../app/blog/blog-image";
import { BlogList, BlogListItem, BlogListNumbered } from "../app/blog/blog-list";
import { BlogQuote } from "../app/blog/blog-quote";
import { Alert } from "./ui/alert/alert";

type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: (props: any) => <BlogImage size="sm" imageUrl={props} />,
  Callout: Alert,
  th: (props: any) => (
    <th {...props} className="text-white font-semibold text-base text-left pb-4" />
  ),
  tr: (props: any) => <tr {...props} className="border-b-[.75px] border-white/10 text-left" />,
  td: (props: any) => (
    <td {...props} className="text-white/70 text-base font-normal py-4 text-left" />
  ),
  a: (props: any) => (
    <a {...props} className="text-white underline hover:text-white/60 text-left" />
  ),
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  ol: (props: any) => BlogListNumbered(props),
  ul: (props: any) => BlogList(props),
  li: (props: any) => BlogListItem(props),
  h1: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient text-white/60" />
  ),
  h2: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient text-white/60" />
  ),
  h3: (props: any) => (
    <h3 {...props} className="text-xl font-medium leading-8 blog-heading-gradient text-white/60" />
  ),
  h4: (props: any) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient text-white/60" />
  ),
  p: (props: any) => (
    <p {...props} className="text-lg font-normal leading-8 text-left text-white/60" />
  ),
  code: (props: any) => (
    <code
      {...props}
      className="inline font-mono text-xs rounded-lg leading-6 xxs:text-xs md:text-base font-normal bg-white/10 text-white px-1.5 py-1 w-full text-nowrap overflow-x-auto"
    />
  ),
  pre: (props: any) => (
    <pre
      {...props}
      className="bg-transparent [&>*]my-6 [&>*]:py-6 [&>*]:px-4 [&>*]:block w-full [&>*]:rounded-xl m-0 p-0"
    />
  ),
  BlogCodeBlock: (props: any) => BlogCodeBlock(props),
};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
