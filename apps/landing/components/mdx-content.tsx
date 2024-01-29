"use client";
import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import Image from "next/image";

import { BlogQuote } from "./blog-quote";
import { BlogList, BlogListItem } from "./blog-list";
import { Alert } from "./ui/alert/alert";
type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: Image,
  Callout: Alert,
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  a: (props: any) => <a {...props} className="text-blue-500" />,
  ul: (props: any) => BlogList(props),
  li: (props: any) => BlogListItem(props),
  h2: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient" />
  ),
  p: (props: any) => <p {...props} className="text-lg font-normal leading-8" />,

};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
