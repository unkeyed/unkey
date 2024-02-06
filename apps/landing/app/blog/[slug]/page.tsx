import { BlogAuthors } from "@/components/blog/blog-authors";
import { Container } from "@/components/container";
// import { FadeIn } from "@/components/landing/fade-in";
import { MdxContent } from "@/components/mdx-content";
// import { PageLinks } from "@/components/landing/page-links";
import { Avatar, AvatarImage } from "@/components/ui/avatar";
// import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, getContentData, getFilePaths, getPost } from "@/lib/mdx-helper";
import { format } from "date-fns";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

type Props = {
  params: { slug: string };
  searchParams: { [key: string]: string | string[] | undefined };
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  // read route params
  const { frontmatter } = await getPost(params.slug);

  if (!frontmatter) {
    return notFound();
  }

  return {
    title: `${frontmatter.title} | Unkey`,
    description: frontmatter.description,
    openGraph: {
      title: `${frontmatter.title} | Unkey`,
      description: frontmatter.description,
      url: `https://unkey.dev/blog/${params.slug}`,
      siteName: "unkey.dev",
    },
    twitter: {
      card: "summary_large_image",
      title: `${frontmatter.title} | Unkey`,
      description: frontmatter.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
}

export const generateStaticParams = async () => {
  const posts = await getFilePaths(BLOG_PATH);
  // Remove file extensions for page paths
  posts.map((path) => path.replace(/\.mdx?$/, "")).map((slug) => ({ params: { slug } }));
  return posts;
};

const BlogArticleWrapper = async ({ params }: { params: { slug: string } }) => {
  const { serialized, frontmatter, headings } = await getPost(params.slug);

  const author = authors[frontmatter.author];
  const _moreArticles = await getContentData({
    contentPath: BLOG_PATH,
    filepath: params.slug,
  });
  console.log(JSON.stringify(serialized));

  return (
    <>
      <Container className="scroll-smooth mb-24">
        <div className="relative mt-16 flex flex-col items-start space-y-8 lg:mt-32 lg:flex-row lg:space-y-0">
          <div className="mx-auto w-full lg:pl-8">
            <h2 className="text-left text-6xl font-medium tracking-tight blog-heading-gradient leading-[72px] pl-24 pr-30">
              {frontmatter.title}
            </h2>
            <p className="my-8 text-left text-white/40 text-lg font-normal leading-8 pl-24 pr-40">
              {frontmatter.description}
            </p>
            <div className="bg-black flex flex-col gap-20">
              <MdxContent source={serialized} />
            </div>
          </div>

          <div className="top-24 flex h-max w-full flex-col justify-end self-start px-4 sm:px-6 lg:sticky lg:w-2/5 lg:px-28 gap-8">
            <div>
              <BlogAuthors author={author} className="w-full mb-8" />

              <p className="text-white/30 text-sm mb-4">Published on</p>
              <h3 className="text-white">{format(new Date(frontmatter.date!), "MMM dd, yyyy")}</h3>
            </div>

            <div className="text-white w-52 overflow-clip max-sm:hidden md:hidden lg:block">
              <p className="text-white/30 text-md">Contents</p>
              <div className="mt-6 overflow-hidden whitespace-nowrap">
                {headings.map((heading) => {
                  return (
                    <div key={`#${heading.slug}`} className="my-4 text-ellipsis text-nowrap ">
                      <a
                        data-level={heading.level}
                        className={
                          heading.level === "two" || heading.level === "one"
                            ? "text-md font-semibold"
                            : "text-sm"
                        }
                        href={`#${heading.slug}`}
                      >
                        {heading.text}
                      </a>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      </Container>
    </>
  );
};

export default BlogArticleWrapper;
