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
import { Post, allPosts } from ".contentlayer/generated";

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
      <BlogContainer className="mt-32 overflow-hidden scroll-smooth ">
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
        <div className="flex flex-col xl:flex-row">
          <div className="mx-6 flex flex-col sm:pl-4 md:px-12 lg:w-10/12 lg:pl-24 xl:mt-12">
            <h1 className="blog-heading-gradient xl:pr-30 pr-0 text-left text-6xl text-[40px] font-medium leading-[56px] tracking-tight sm:pt-8 sm:text-[56px] sm:leading-[72px] xl:mt-0 xl:w-3/4">
              {post.title}
            </h1>
            <p className="mt-10 text-left text-lg font-normal leading-8 text-white/40 ">
              {post.description}
            </p>
          </div>
          <div className="mt-6 flex w-full flex-col justify-start p-0 pl-6 lg:w-2/12">
            <div className="mb-8 flex w-full flex-row items-start gap-2 md:ml-12 lg:ml-24 lg:gap-12 xl:ml-0 xl:flex-col">
              <BlogAuthors author={author} className="mb-0 mt-0 w-40 sm:ml-4 lg:w-full" />
              <div className="mt-0 flex w-full flex-col">
                <p className="mb-0 text-nowrap text-white/30">Published on</p>
                <time dateTime={post.date} className="mt-8 text-nowrap pt-1 text-white xl:pt-0">
                  {format(parseISO(post.date), "MMM dd, yyyy")}
                </time>
              </div>
            </div>
          </div>
        </div>
        <div className="mb-40 flex ">
          <div className="flex w-full flex-col gap-12 xl:w-10/12 ">
            <div className="flex ">
              <Frame className="mx-6 h-full w-full px-0 shadow-sm xl:mx-12" size="lg">
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
          <div className="hidden w-2/12 pt-12 text-white xl:ml-6 xl:flex xl:flex-col">
            <p className="text-md text-white/30">Contents</p>

            <div className="flex flex-col">
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
