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
    <th {...props} className="pb-4 text-base font-semibold text-left text-white" />
  ),
  tr: (props: any) => <tr {...props} className="border-b-[.75px] border-white/10 text-left" />,
  td: (props: any) => (
    <td {...props} className="py-4 text-base font-normal text-left text-white/70" />
  ),
  a: (props: any) => (
    <a
      {...props}
      aria-label="Link"
      className="text-left text-white underline hover:text-white/60"
    />
  ),
  blockquote: (props: any) => BlogQuote(props),
  BlogQuote: (props: any) => BlogQuote(props),
  ol: (props: any) => BlogListNumbered(props),
  ul: (props: any) => BlogList(props),
  li: (props: any) => BlogListItem(props),
  h1: (props: any) => (
    <h2
      {...props}
      className="text-2xl font-medium leading-8 blog-heading-gradient text-white/60 scroll-mt-20"
    />
  ),
  h2: (props: any) => (
    <h2
      {...props}
      className="text-2xl font-medium leading-8 blog-heading-gradient text-white/60 scroll-mt-20"
    />
  ),
  h3: (props: any) => (
    <h3
      {...props}
      className="text-xl font-medium leading-8 blog-heading-gradient text-white/60 scroll-mt-20"
    />
  ),
  h4: (props: any) => (
    <h4 {...props} className="text-lg font-medium leading-8 blog-heading-gradient text-white/60" />
  ),
  p: (props: any) => (
    <p {...props} className="text-lg font-normal leading-8 text-left text-white/60" />
  ),
  code: (props: any) => (
    <code
      className="px-2 py-1 font-medium text-gray-600 border border-gray-200 rounded-md bg-gray-50 before:hidden after:hidden"
      {...props}
    />
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
