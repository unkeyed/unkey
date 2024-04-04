import { BlogAuthors } from "@/components/blog/blog-authors";
import { BlogContainer } from "@/components/blog/blog-container";
import { SuggestedBlogs } from "@/components/blog/suggested-blogs";
import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";
import { MDX } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
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
        <div className="w-full -z-40 overflow-clip">
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
            className="overflow-hidden sm:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={2}
            className="overflow-hidden sm:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={7}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={2}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={5}
            className="overflow-hidden sm:hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={0}
            className="overflow-hidden sm:hidden md:block"
          />
        </div>
        <div className="overflow-hidden -z-40">
          <TopRightShiningLight />
        </div>
        <div className="flex gap-8 mx-auto lg:gap-16">
          <div className="flex flex-col items-center w-full xl:w-9/12">
            <div className="prose sm:prose-sm md:prose-md">
              <h1 className="blog-heading-gradient text-left text-4xl font-medium leading-[56px] tracking-tight  sm:text-5xl sm:leading-[72px]">
                {post.title}
              </h1>
              <p className="text-lg font-normal leading-8 text-white/40 ">{post.description}</p>
            </div>
          </div>
          <div className="flex flex-col justify-start w-full p-0 pl-6 mt-6 lg:w-2/12">
            <div className="flex flex-row items-start w-full gap-2 mb-8 md:ml-12 lg:ml-24 lg:gap-12 xl:ml-0 xl:flex-col">
              <BlogAuthors author={author} className="w-40 mt-0 mb-0 sm:ml-4 lg:w-full" />
              <div className="flex flex-col w-full mt-0">
                <p className="mb-0 text-nowrap text-white/30">Published on</p>
                <time dateTime={post.date} className="pt-1 mt-8 text-white text-nowrap xl:pt-0">
                  {format(parseISO(post.date), "MMM dd, yyyy")}
                </time>
              </div>
            </div>
          </div>
        </div>
        <div className="flex gap-8 mx-auto mb-40 lg:gap-16">
          <div className="flex flex-col items-center w-full gap-12 xl:w-9/12 ">
            <Frame className="w-full h-full px-0 overflow-clip" size="lg">
              <Image
                src={post.image ?? "/images/blog-images/defaultBlog.png"}
                width={1200}
                height={860}
                alt={post.title}
              />
            </Frame>
            <div className="flex flex-col gap-4 prose md:gap-8 lg:gap-12 sm:prose-sm md:prose-md ">
              <MDX code={post.body.code} />
            </div>
          </div>
          <div className="hidden w-3/12 pt-12 text-white xl:ml-6 xl:flex xl:flex-col">
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
