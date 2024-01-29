"use client";
import { authors } from "@/content/blog/authors";
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
  blockquote: (props: any) => BlogQuote(props),
  a: (props: any) => <a {...props} className="text-blue-500" />,

  h2: (props: any) => (
    <h2 {...props} className="text-2xl font-medium leading-8 blog-heading-gradient" />
  ),
  p: (props: any) => (
    <p
      {...props}
      className="text-lg font-normal leading-8 xl:pl-20 max-sm:pl-4 sm-pl-4 md:pl-8 lg:mt-18 xl:mt-20"
    />
  ),
};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
