import { useMDXComponent } from "@content-collections/mdx/react";
import type { ImageProps } from "next/image";
import Image from "next/image";
import type { DetailedHTMLProps, ImgHTMLAttributes, JSX } from "react";
import { BlogCodeBlock, BlogCodeBlockSingle } from "./blog/blog-code-block";
import { BlogList, BlogListItem, BlogListNumbered, type BlogListProps } from "./blog/blog-list";
import { BlogQuote, type BlogQuoteProps } from "./blog/blog-quote";
import { ImageWithBlur } from "./image-with-blur";
import { Alert } from "./ui/alert/alert";

export const MdxComponents = {
  Image: (props: ImageProps) =>
    props.width || props.fill ? (
      <Image
        {...props}
        placeholder="blur"
        blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
      />
    ) : (
      <Image
        {...props}
        placeholder="blur"
        width={1920}
        height={1080}
        blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
      />
    ),
  img: (props: DetailedHTMLProps<ImgHTMLAttributes<HTMLImageElement>, HTMLImageElement>) => (
    <img src={props.src} alt={props.src} />
  ),
  Callout: Alert,
  th: (props: JSX.IntrinsicAttributes) => (
    <th {...props} className="pb-4 text-base font-semibold text-left text-white" />
  ),
  tr: (props: JSX.IntrinsicAttributes) => (
    <tr {...props} className="border-b-[.75px] border-white/10 text-left" />
  ),
  td: (props: JSX.IntrinsicAttributes) => (
    <td {...props} className="py-4 text-base font-normal text-left text-white/70" />
  ),
  a: (props: JSX.IntrinsicAttributes) => (
    <a
      {...props}
      aria-label="Link"
      className="text-left text-white underline hover:text-white/60"
    />
  ),
  blockquote: (props: BlogQuoteProps) => BlogQuote(props),
  BlogQuote: (props: BlogQuoteProps) => BlogQuote(props),
  ol: (props: BlogListProps) => BlogListNumbered(props),
  ul: (props: BlogListProps) => BlogList(props),
  li: (props: BlogListProps) => BlogListItem(props),
  h1: (props: any) => (
    <h2
      {...props}
      className="text-2xl font-medium leading-8 blog-heading-gradient text-white/60 scroll-mt-20"
    />
  ),
  h2: (props: JSX.IntrinsicAttributes) => (
    <h2
      {...props}
      className="text-2xl font-medium leading-8 blog-heading-gradient text-white/60 scroll-mt-20"
    />
  ),
  h3: (props: JSX.IntrinsicAttributes) => (
    <h3
      {...props}
      className="text-xl font-medium leading-8 blog-heading-gradient text-white/60 scroll-mt-20"
    />
  ),
  h4: (props: JSX.IntrinsicAttributes) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient text-white/60" />
  ),
  p: (props: JSX.IntrinsicAttributes) => (
    <p {...props} className="text-lg font-normal leading-8 text-left text-white/60" />
  ),
  code: (props: JSX.IntrinsicAttributes) => (
    <code
      className="px-2 py-1 font-medium text-gray-600 border border-gray-200 rounded-md bg-gray-50 before:hidden after:hidden"
      {...props}
    />
  ),
  pre: (props: JSX.IntrinsicAttributes) => <BlogCodeBlockSingle {...props} />,
  BlogCodeBlock,
};

interface MDXProps {
  code: string;
}

export function MDX({ code }: MDXProps) {
  const Component = useMDXComponent(code);
  return (
    <Component
      components={{
        ...MdxComponents,
      }}
    />
  );
}
