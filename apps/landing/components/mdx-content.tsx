"use client";
import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import { BlogCodeBlock } from "./blog-code-block";
import { BlogList, BlogListItem, BlogListNumbered } from "./blog-list";
import { BlogQuote } from "./blog-quote";
import { BlogImage } from "./blog/blog-image";
import { Alert } from "./ui/alert/alert";

type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: (props: any) => <BlogImage size="lg" imageUrl={props} />,
  Callout: Alert,
  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60" />,
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  ol: (props: any) => BlogListNumbered(props),
  ul: (props: any) => BlogList(props),
  li: (props: any) => BlogListItem(props),
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
  code: (props: any) => BlogCodeBlock(props),
  BlogCodeBlock: (props: any) => BlogCodeBlock(props),
  //pre: (props: any) => BlogCodeBlock(props),
  // code: (props: any) => (
  //   <code
  //     {...props}
  //     className="border-b-[20px] border-t-[.5px] border-[rgba(255,255,255,0.1)] bg-transparent"
  //   />
  // ),
};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
