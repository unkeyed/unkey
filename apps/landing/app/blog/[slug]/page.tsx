import { BlogAuthors } from "@/components/blog/blog-authors";
import { BlogContainer } from "@/components/blog/blog-container";
import { SuggestedBlogs } from "@/components/blog/suggested-blogs";
import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";
import { MDX } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format, parseISO } from "date-fns";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import { type Post, allPosts } from ".contentlayer/generated";

interface Heading {
  level: number | undefined;
  text: string;
  slug: string;
}

export const generateStaticParams = async () =>
  allPosts.map((post) => ({ slug: post._raw.flattenedPath.replace("blog/", "") }));

export const generateMetadata = ({ params }: { params: { slug: string } }) => {
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
      url: `https://unkey.dev/${post._raw.flattenedPath}`,
      siteName: "unkey.dev",
      type: "article",
      article: {
        publishedTime: format(parseISO(post.date), "yyyy-MM-dd"),
        modifiedTime: format(parseISO(post.date), "yyyy-MM-dd"),
        tags: post.tags,
      },
      ogImage: {
        url: `https://unkey.dev${post.image}`,
        width: 800,
        height: 600,
      },
    },
    twitter: {
      card: "summary_large_image",
      title: `${post.title} | Unkey`,
      description: post.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
};

const BlogArticleWrapper = async ({ params }: { params: { slug: string } }) => {
  const post = allPosts.find((post) => post._raw.flattenedPath === `blog/${params.slug}`) as Post;
  if (!post) {
    notFound();
  }

  const author = authors[post.author];
  return (
    <>
      <div className="container mx-auto mt-32 overflow-hidden scroll-smooth ">
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
            className="overflow-hidden hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={2}
            className="overflow-hidden hidden md:block"
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
        <div className="flex flex-col">
          <div className="flex flex-col items-start justify-between w-full lg:flex-row sm:mx-6 lg:pl-4">
            <div className="w-fit justify-left mr-0 pr-0">
              <div className="prose sm:prose-sm md:prose-md">
                <div className="flex items-center gap-5 mb-8 font-medium text-xl leading-8">
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
                <p className="mt-8 text-lg font-normal leading-8 not-prose text-white/60 sm:pr-8">
                  {post.description}
                </p>
              </div>
            </div>

            <div className="w-full h-full prose sm:prose-sm md:prose-md lg:flex lg:flex-col lg:w-1/4 mr-2">
              <div className="flex flex-row items-start lg:w-full w-fit mt-4 gap-16 md:mt-16 not-prose lg:mt-0 lg:flex-col h-full">
                <div className="flex flex-col gap-4 lg:gap-4 w-full lg:mt-12">
                  <p className="text-white/40">Written by</p>
                  <div className="flex flex-col ">
                    <div className="flex items-center sm:gap-0 gap-2">
                      <Avatar className="w-10 h-10">
                        <AvatarImage
                          alt={author.name}
                          src={author.image.src}
                          width={12}
                          height={12}
                          className="w-full"
                        />
                        <AvatarFallback />
                      </Avatar>
                      <p className="text-white text-nowrap lg:block ml-2">{author.name}</p>
                    </div>
                  </div>
                </div>

                <div className="flex flex-col gap-4 not-prose lg:gap-2">
                  <p className="mb-0 text-nowrap text-white/30">Published on</p>
                  <time
                    dateTime={post.date}
                    className="inline-flex items-center h-10 text-white text-nowrap"
                  >
                    {format(parseISO(post.date), "MMM dd, yyyy")}
                  </time>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div className="flex justify-between w-full gap-8 mx-auto mt-12 mb-40 lg:gap-16 ">
          <div className="flex flex-col w-full gap-8 lg:gap-16 lg:w-3/4 ">
            <Frame className="px-0 overflow-clip" size="lg">
              <Image
                src={post.image ?? "/images/blog-images/defaultBlog.png"}
                width={1200}
                height={860}
                alt={post.title}
              />
            </Frame>
            <div className="xs:prose:xs sm:prose-sm md:prose-md text-white/60 sm:mx-6 prose-strong:text-white/90 prose-code:text-white/80 prose-code:bg-white/10 prose-code:px-2 prose-code:py-1 prose-code:border-white/20 prose-code:rounded-md">
              <MDX code={post.body.code} />
            </div>
          </div>
          <div className="hidden w-1/4 text-white lg:flex lg:flex-col">
            {post.tableOfContents.length !== 0 ? (
              <>
                <p className="text-md text-white/50">Contents</p>
                <ul className="relative flex flex-col gap-2 mt-2 overflow-hidden">
                  {post.tableOfContents.map((heading: Heading) => {
                    return (
                      <Link
                        key={`#${heading.slug}`}
                        data-level={heading.level}
                        className={cn({
                          "text-sm  font-medium mt-4 text-transparent bg-clip-text bg-gradient-to-r from-white to-white/60":
                            heading.level === 1 || heading.level === 2,
                          "text-sm ml-4 leading-8 text-transparent bg-clip-text bg-gradient-to-r from-white/50 to-white/40":
                            heading.level === 3,
                        })}
                        href={`#${heading.slug}`}
                      >
                        {heading.text}
                      </Link>
                    );
                  })}
                </ul>
              </>
            ) : null}
            <div className="flex flex-col mt-10">
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
