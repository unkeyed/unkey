import { Container } from "@/components/container";

import { BlogHeading, BlogSubTitle, BlogTitle } from "@/components/blog-heading";
import { MdxContent } from "@/components/mdx-content";
import { authors } from "@/content/blog/authors";
import type { Metadata } from "next";

import { notFound } from "next/navigation";

import { BlogHero } from "@/components/blog-hero";
import { BLOG_PATH, getContentData, getFilePaths, getPost } from "@/lib/mdx-helper";

type Props = {
  params: { slug: string };
  searchParams: { [key: string]: string | string[] | undefined };
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  // read route params
  const { frontmatter } = await getPost("how-unkey-treats-marketing");

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
  const { serialized, frontmatter } = await getPost("how-unkey-treats-marketing");

  const author = authors[frontmatter.author];
  const _moreArticles = await getContentData({
    contentPath: BLOG_PATH,
    filepath: params.slug,
  });

  return (
    <>
      <Container className="scroll-smooth my-8">
        <BlogHero
          label={"Product"}
          imageUrl="/images/blog-images/ai-post/create-api.png"
          title={frontmatter.title}
          subTitle={frontmatter.description}
          author={author.name}
          publishDate={new Date(frontmatter.date).toDateString()}
        />
        <div className="relative mt-16 flex flex-col items-start space-y-8 lg:mt-32 lg:flex-row lg:space-y-0">
          <div className="mx-auto w-full lg:pl-8 ">
            <BlogHeading>
              <BlogTitle>{frontmatter.title}</BlogTitle>
              <BlogSubTitle>{frontmatter.description}</BlogSubTitle>
            </BlogHeading>

            <div className="prose text-white/60 font-normal text-lg leading-8 mx-auto w-full pt-20">
              <MdxContent source={serialized} />
            </div>
          </div>

          <div className="top-24 flex h-max w-full flex-col justify-end self-start px-4 sm:px-6 lg:sticky lg:w-2/5 lg:px-8">
            <div className="mx-auto flex items-center justify-start gap-4 border-y-0 p-2 md:mx-0 md:border-b md:border-b-gray-200">
              <div className="text-sm text-white/60">
                <div className="font-semibold">{author.name}</div>
              </div>
            </div>
            {
              <div className="hidden md:block">
                <h3 className="mb-4 mt-8 text-lg font-bold uppercase tracking-wide text-white/60">
                  Table of Contents
                </h3>
              </div>
            }
          </div>
        </div>
      </Container>
    </>
  );
};

export default BlogArticleWrapper;
