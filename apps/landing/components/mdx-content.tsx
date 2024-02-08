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
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient" />
  ),
  h3: (props: any) => (
    <h3 {...props} className="text-xl font-medium leading-8 blog-heading-gradient" />
  ),
  h4: (props: any) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient" />
  ),
  p: (props: any) => (
    <p {...props} className="text-lg font-normal leading-8 text-white/60 text-left" />
  ),
  code: (props: any) => (
    <span
      {...props}
      className="font-mono rounded-md code-inline-gradient text-white/50 px-3 pt-0.5 pb-1 mx-1"
    />
  ),
  BlogCodeBlock: (props: any) => BlogCodeBlock(props),
};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
