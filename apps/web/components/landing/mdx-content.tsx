"use client"; // This is required!
import { MDXRemote, type MDXRemoteSerializeResult } from "next-mdx-remote";
import Image from "next/Image";

type MdxContentProps = {
  source: MDXRemoteSerializeResult;
};

/** Custom components here!*/
const MdxComponents = {
  Image: Image,
};

export function MdxContent({ source }: MdxContentProps) {
  return <MDXRemote {...source} components={MdxComponents} />;
}
