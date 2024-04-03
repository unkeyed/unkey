import { useMDXComponent } from "next-contentlayer/hooks";
import { BlogCodeBlock, BlogCodeBlockSingle } from "./blog/blog-code-block";
import { BlogImage } from "./blog/blog-image";
import { BlogList, BlogListItem, BlogListNumbered } from "./blog/blog-list";
import { BlogQuote } from "./blog/blog-quote";
import { Alert } from "./ui/alert/alert";
/** Custom components here!*/
export const MdxComponents = {
  Image: (props: any) => <BlogImage size="sm" imageUrl={props} />,
  img: (props: any) => <BlogImage size="sm" imageUrl={props} />,
  Callout: Alert,
  th: (props: any) => (
    <th {...props} className="pb-4 text-left text-base font-semibold text-white" />
  ),
  tr: (props: any) => <tr {...props} className="border-b-[.75px] border-white/10 text-left" />,
  td: (props: any) => (
    <td {...props} className="py-4 text-left text-base font-normal text-white/70" />
  ),
  a: (props: any) => (
    <a {...props} className="text-left text-white underline hover:text-white/60" />
  ),
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  ol: (props: any) => BlogListNumbered(props),
  ul: (props: any) => BlogList(props),
  li: (props: any) => BlogListItem(props),
  h1: (props: any) => (
    <h2 {...props} className="blog-heading-gradient text-2xl font-medium leading-8 text-white/60" />
  ),
  h2: (props: any) => (
    <h2 {...props} className="blog-heading-gradient text-2xl font-medium leading-8 text-white/60" />
  ),
  h3: (props: any) => (
    <h3 {...props} className="blog-heading-gradient text-xl font-medium leading-8 text-white/60" />
  ),
  h4: (props: any) => (
    <h4 {...props} className="blog-heading-gradient text-lg font-medium leading-8 text-white/60" />
  ),
  p: (props: any) => (
    <p {...props} className="text-left text-lg font-normal leading-8 text-white/60" />
  ),
  pre: BlogCodeBlockSingle,
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
