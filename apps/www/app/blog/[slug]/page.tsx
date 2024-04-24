import { SuggestedBlogs } from "@/components/blog/suggested-blogs";
import { CTA } from "@/components/cta";
import { MDX } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format, parseISO } from "date-fns";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { type Post, allPosts } from ".contentlayer/generated";

interface Heading {
  level: number | undefined;
  text: string;
  slug: string;
}

export const generateStaticParams = async () =>
  allPosts.map((post) => ({
    slug: post._raw.flattenedPath.replace("blog/", ""),
  }));

export function generateMetadata({ params }: { params: { slug: string } }): Metadata {
  const post = allPosts.find((post) => post._raw.flattenedPath === `blog/${params.slug}`);
  if (!post) {
    notFound();
  }
  return {
    title: `${post.title} | Unkey`,
    description: post.description,
    openGraph: {
      title: `${post.title} | Unkey`,
      description: post.description,
      url: `https://unkey.com/${post._raw.flattenedPath}`,
      siteName: "unkey.com",
      type: "article",
      images: {
        url: `https://unkey.com${post.image}`,
        width: 1200,
        height: 800,
      },

      publishedTime: format(parseISO(post.date), "yyyy-MM-dd"),
      modifiedTime: format(parseISO(post.date), "yyyy-MM-dd"),
      tags: post.tags,
    },
    twitter: {
      card: "summary_large_image",
      title: `${post.title} | Unkey`,
      description: post.description,
      site: "@unkeydev",
      creator: "@unkeydev",
      images: {
        url: `https://unkey.com${post.image}`,
        width: 1200,
        height: 800,
      },
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
}

const BlogArticleWrapper = async ({ params }: { params: { slug: string } }) => {
  const post = allPosts.find((post) => post._raw.flattenedPath === `blog/${params.slug}`) as Post;
  if (!post) {
    notFound();
  }

  const author = authors[post.author];
  return (
    <>
      <div className="container pt-48 mx-auto sm:overflow-hidden md:overflow-visible scroll-smooth ">
        <div>
          <TopLeftShiningLight className="hidden h-full -z-40 sm:block" />
        </div>
        <div className="w-full h-full overflow-hidden -z-20">
          <MeteorLinesAngular
            number={1}
            xPos={0}
            speed={10}
            delay={5}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={0}
            speed={10}
            delay={0}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={7}
            className="overflow-hidden md:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={2}
            className="overflow-hidden md:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={7}
            className="hidden overflow-hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={2}
            className="hidden overflow-hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={5}
            className="hidden overflow-hidden lg:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={0}
            className="hidden overflow-hidden lg:block"
          />
        </div>
        <div className="overflow-hidden -z-40">
          <TopRightShiningLight />
        </div>
        <div className="flex flex-row w-full">
          <div className="flex flex-col w-full lg:w-3/4">
            <div className="prose sm:prose-sm md:prose-md sm:mx-6">
              <div className="flex items-center gap-5 p-0 m-0 mb-8 text-xl font-medium leading-8">
                <Link href="/blog">
                  <span className="text-transparent bg-gradient-to-r bg-clip-text from-white to-white/60 ">
                    Blog
                  </span>
                </Link>
                <span className="text-white/40">/</span>
                <Link href={`/blog?tag=${post.tags?.at(0)}`}>
                  <span className="text-transparent capitalize bg-gradient-to-r bg-clip-text from-white to-white/60">
                    {post.tags?.at(0)}
                  </span>
                </Link>
              </div>

              <h1 className="not-prose blog-heading-gradient text-left text-4xl font-medium leading-[56px] tracking-tight  sm:text-5xl sm:leading-[72px]">
                {post.title}
              </h1>
              <p className="mt-8 text-lg font-medium leading-8 not-prose text-white/60 lg:text-xl">
                {post.description}
              </p>
              <div className="flex flex-row gap-8 sm:mt-12 md:gap-16 lg:hidden justify-stretch ">
                <div className="flex flex-col h-full">
                  <p className="text-white/50">Written by</p>
                  <div className="flex flex-row h-full">
                    <Avatar className="flex items-center my-auto">
                      <AvatarImage
                        alt={author.name}
                        src={author.image.src}
                        width={12}
                        height={12}
                        className="w-full h-full"
                      />
                      <AvatarFallback />
                    </Avatar>
                    <p className="flex items-center justify-center p-0 pt-2 m-0 ml-2 text-white text-nowrap">
                      {author.name}
                    </p>
                  </div>
                </div>
                <div className="flex flex-col h-full">
                  {" "}
                  <p className="text-xs text-nowrap text-white/50">Published on</p>
                  <div className="flex mt-2 sm:mt-6">
                    <time
                      dateTime={post.date}
                      className="inline-flex items-center text-white text-nowrap"
                    >
                      {format(parseISO(post.date), "MMM dd, yyyy")}
                    </time>
                  </div>
                </div>
              </div>
            </div>
            <div className="mt-12 prose-sm lg:pr-24 md:prose-md text-white/60 sm:mx-6 prose-strong:text-white/90 prose-code:text-white/80 prose-code:bg-white/10 prose-code:px-2 prose-code:py-1 prose-code:border-white/20 prose-code:rounded-md prose-pre:p-0 prose-pre:m-0 prose-pre:leading-6">
              <MDX code={post.body.code} />
            </div>
          </div>

          <div className="items-start hidden h-full gap-4 pt-8 space-y-4 prose lg:sticky top-24 lg:w-1/4 not-prose lg:mt-12 lg:flex lg:flex-col">
            <div className="flex flex-col gap-4 not-prose lg:gap-2">
              <p className="text-sm text-white/50">Written by</p>
              <div className="flex flex-col h-full gap-2 mt-1 xl:flex-row">
                <Avatar className="w-10 h-10 mr-4">
                  <AvatarImage
                    alt={author.name}
                    src={author.image.src}
                    width={12}
                    height={12}
                    className="w-full"
                  />
                  <AvatarFallback />
                </Avatar>
                <p className="my-auto text-white text-nowrap">{author.name}</p>
              </div>
            </div>
            <div className="flex flex-col gap-4 mt-4 not-prose lg:gap-2">
              <p className="text-sm text-nowrap text-white/50">Published on</p>
              <time
                dateTime={post.date}
                className="inline-flex items-center h-10 text-white text-nowrap"
              >
                {format(parseISO(post.date), "MMM dd, yyyy")}
              </time>
            </div>
            {post.tableOfContents.length !== 0 ? (
              <div className="flex flex-col gap-4 not-prose lg:gap-2">
                <p className="text-sm prose text-nowrap text-white/50">Contents</p>
                <ul className="relative flex flex-col gap-1 overflow-hidden">
                  {post.tableOfContents.map((heading: Heading) => {
                    return (
                      <li key={`#${heading.slug}`}>
                        <Link
                          data-level={heading.level}
                          className={cn({
                            "text-md font-medium mt-4 text-transparent bg-clip-text bg-gradient-to-r from-white  to-white/70 truncate":
                              heading.level === 1 || heading.level === 2,
                            "text-sm ml-4 leading-8 text-transparent bg-clip-text bg-gradient-to-r from-white/60  to-white/50 truncate":
                              heading.level === 3 || heading.level === 4,
                          })}
                          href={`#${heading.slug}`}
                        >
                          {heading.text}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            ) : null}
            <div className="flex flex-col mt-4">
              <p className="pt-10 text-md text-white/50">Suggested</p>
              <div>
                <SuggestedBlogs currentPostSlug={post.url} />
              </div>
            </div>
          </div>
        </div>
        <CTA />
      </div>
    </>
  );
};

export default BlogArticleWrapper;
