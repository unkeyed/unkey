import { BlogAuthors } from "@/components/blog/blog-authors";
import { BlogContainer } from "@/components/blog/blog-container";
import { SuggestedBlogs } from "@/components/blog/suggested-blogs";
import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";
import { MDX } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { format, parseISO } from "date-fns";
import Image from "next/image";
import { notFound } from "next/navigation";
import { type Post, allPosts } from ".contentlayer/generated";

interface Heading {
  level: "one" | "two" | "three";
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
      <BlogContainer className="w-[1440px] mt-32 overflow-hidden scroll-smooth ">
        <div>
          <TopLeftShiningLight className="-z-40 hidden h-full sm:block" />
        </div>
        <div className="-z-40 w-full overflow-clip">
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
        <div className="-z-40 overflow-hidden">
          <TopRightShiningLight />
        </div>
        <div className="flex flex-col xl:flex-row pl-6 sm:pl-12 md:pl-[72px] lg:pl-28 xl:pl-0 xl:ml-0 w-full">
          <div className="flex flex-col lg:w-8/12 xl:mt-12 xl:ml-32 w-full ">
            <h1 className="text-left pt-4 sm:pt-8 text-[40px] sm:text-[56px] text-6xl font-medium tracking-tight blog-heading-gradient leading-[56px] sm:leading-[72px] xl:w-3/4">
              {post.title}
            </h1>
            <p className="mt-10 text-left text-lg font-normal leading-8 text-white/40">
              {post.description}
            </p>
          </div>
          <div className="sm:mt-8 mt-12 xl:mt-24 flex w-full flex-col justify-start lg:w-4/12 xl:pl-20">
            <BlogAuthors author={author} className="mb-0 mt-0 w-40 lg:ml-0 lg:w-full" />

            <p className="mt-4 text-nowrap text-white/30">Published on</p>
            <time dateTime={post.date} className="mt-2 text-nowrap pt-1 text-white xl:pt-0">
              {format(parseISO(post.date), "MMM dd, yyyy")}
            </time>
          </div>
        </div>
        <div className="mb-40 flex mt-12 ">
          <div className="flex w-full flex-col gap-12 xl:w-9/12 ">
            <div className="flex mx-2 md:px-12">
              <Frame className="overflow-clip h-full w-full px-0" size="lg">
                <Image
                  src={post.image ?? "/images/blog-images/defaultBlog.png"}
                  width={1200}
                  height={860}
                  alt={post.title}
                />
              </Frame>
            </div>
            <div className="mx-6 flex flex-col gap-12 sm:px-4 md:px-12 lg:px-24">
              <MDX code={post.body.code} />
            </div>
          </div>
          <div className="hidden w-3/12 pt-12 text-white xl:ml-6 xl:flex xl:flex-col">
            {post.tableOfContents.length !== 0 && (
              <>
                <p className="text-md text-white/30">Contents</p>
                <div className="relative mt-2 overflow-hidden ">
                  {/* <div className="absolute top-0 left-0 z-20 w-full h-full bg-gradient-to-r from-transparent via-[#010101]/30 to-[#010101]/100" /> */}
                  {post.tableOfContents.map((heading: Heading) => {
                    return (
                      <div
                        key={`#${heading.slug}`}
                        className="blog-heading-gradient z-0 my-8 text-ellipsis"
                      >
                        <a
                          data-level={heading.level}
                          className={
                            heading.level === "two" ||
                            heading.level === "one" ||
                            heading.level === "three"
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
              </>
            )}
            <div className="flex flex-col mr-12">
              <p className="text-md pt-10 text-white/30">Suggested</p>
              <div>
                <SuggestedBlogs currentPostSlug={post.url} />
              </div>
            </div>
          </div>
        </div>
        <CTA />
      </BlogContainer>
    </>
  );
};

export default BlogArticleWrapper;
