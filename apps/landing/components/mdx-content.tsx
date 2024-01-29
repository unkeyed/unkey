"use client";
import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import Image from "next/image";
import { BlogQuote } from "./blog-quote";
import { Alert } from "./ui/alert/alert";
type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: Image,
  Callout: Alert,
  Blockquote: BlogQuote,
  a: (props: any) => <a {...props} className="text-blue-500" />,

  h2: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient" />
  ),
  p: (props: any) => <p {...props} className="text-lg font-normal leading-8" />,
};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
