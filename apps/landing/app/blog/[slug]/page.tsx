import { Container } from "@/components/container";
// import { FadeIn } from "@/components/landing/fade-in";
import { MdxContent } from "@/components/mdx-content";
// import { PageLinks } from "@/components/landing/page-links";
import { Avatar, AvatarImage } from "@/components/ui/avatar";
// import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { authors } from "@/content/blog/authors";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { BLOG_PATH, getContentData, getFilePaths, getPost } from "@/lib/mdx-helper";

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

  return (
    <>
      <Container className="scroll-smooth">
        <div className="relative mt-16 flex flex-col items-start space-y-8 lg:mt-32 lg:flex-row lg:space-y-0">
          <div className="mx-auto w-full lg:pl-8">
            <h2 className="text-center text-3xl font-bold tracking-tight text-gray-900 sm:text-6xl">
              {frontmatter.title}
            </h2>
            <p className="border- my-8 text-center text-gray-500">{frontmatter.description}</p>
            <div className="prose prose-neutral dark:prose-invert prose-img:rounded-lg prose-img:border prose-img:border-border mx-auto w-full">
              <MdxContent source={serialized} />
            </div>
          </div>

          <div className="top-24 flex h-max w-full flex-col justify-end self-start px-4 sm:px-6 lg:sticky lg:w-2/5 lg:px-8">
            <div className="mx-auto flex items-center justify-start gap-4 border-y-0 p-2 md:mx-0 md:border-b md:border-b-gray-200">
              <Avatar className="h-14 w-14 justify-items-start">
                <AvatarImage src={author.image?.src} alt={author.name} />
              </Avatar>
              <div className="text-sm text-gray-950">
                <div className="font-semibold">{author.name}</div>
              </div>
            </div>
            {
              <div className="hidden md:block">
                <h3 className="mb-4 mt-8 text-lg font-bold uppercase tracking-wide text-gray-600">
                  Table of Contents
                </h3>

                <div className="p-4">
                  {headings.map((heading) => {
                    return (
                      <div key={`#${heading.slug}`} className="my-2">
                        <a
                          data-level={heading.level}
                          className={
                            heading.level === "two" || heading.level === "one"
                              ? "text-md font-semibold"
                              : "ml-4 text-sm"
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
            }
          </div>
        </div>
      </Container>
    </>
  );
};

export default BlogArticleWrapper;
