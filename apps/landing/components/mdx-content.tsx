"use client";
import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import Image from "next/image";
import { BlogCodeBlock } from "./blog-code-block";
import { BlogList, BlogListItem } from "./blog-list";
import { BlogQuote } from "./blog-quote";
import { Alert } from "./ui/alert/alert";

type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: Image,
  Callout: Alert,
  a: (props: any) => <a {...props} className="text-white underline hover:text-white/60" />,
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  ul: (props: any) => BlogList(props),
  li: (props: any) => BlogListItem(props),
  h2: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient" />
  ),
  p: (props: any) => <p {...props} className="text-lg font-normal leading-8" />,
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
