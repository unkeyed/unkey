import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";
import { MdxContent } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, getContentData, getFilePaths, getMeta, getPost } from "@/lib/mdx-helper";
import { format } from "date-fns";
import type { Metadata } from "next";
import Image from "next/image";
import { notFound } from "next/navigation";
import { BlogAuthors } from "../blog-authors";
import { BlogContainer } from "../blog-container";
import { SuggestedBlogs } from "../suggested-blogs";

type Props = {
  params: { slug: string };
  searchParams: { [key: string]: string | string[] | undefined };
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { frontmatter } = await getMeta(params.slug);

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
  const paths = posts
    .filter((post) => post.includes(".mdx"))
    .map((post) => {
      return {
        slug: post.replace(".mdx", ""),
      };
    });
  return paths;
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
      <BlogContainer className="max-w-[1440px] mt-32 overflow-hidden scroll-smooth ">
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
          <div className="flex flex-col mx-6 sm:pl-4 lg:pl-24 md:px-12 lg:w-10/12 xl:mt-12">
            <div className="text-white text-[20px] flex">
              {frontmatter.tags && (
                <>
                  <p className="blog-breadcrumb-gradient">Blog</p>
                  <span className="mx-4 text-white/30">/</span>
                  <p className="capitalize blog-breadcrumb-gradient">
                    {frontmatter?.tags?.split(" ")[0]}
                  </p>
                </>
              )}
            </div>
            <h1 className="text-left pt-4 sm:pt-8 text-[40px] sm:text-[56px] text-6xl font-medium tracking-tight blog-heading-gradient leading-[56px] sm:leading-[72px] pr-0 xl:pr-30 xl:w-3/4">
              {frontmatter.title}
            </h1>
            <p className="mt-10 text-left text-lg font-normal leading-8 text-white/40 ">
              {frontmatter.description}
            </p>
          </div>
          <div className="mt-6 flex w-full flex-col justify-start p-0 pl-0 ml-0 lg:w-2/12">
            <div className="mb-8 flex w-full flex-row items-start gap-2 md:ml-12 lg:ml-24 lg:gap-12 xl:ml-0 xl:pl-0 xxl:ml-0 xxl:pl-0 xl:flex-col">
              <BlogAuthors
                author={author}
                className="mb-0 mt-0 w-40 sm:ml-4 lg:ml-0 lg:w-full xl:ml-0 xl:pl-0"
              />
              <div className="mt-0 flex w-full flex-col">
                <p className="mb-0 ml-0 pl-0 text-nowrap text-white/30">Published on</p>
                <p className="mt-8 text-nowrap pt-1 text-white xl:pt-0">
                  {format(new Date(frontmatter.date!), "MMM dd, yyyy")}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div className="mb-40 flex ">
          <div className="flex w-full flex-col gap-12 xl:w-10/12 ">
            <div className="flex px-12">
              <Frame className="overflow-clip h-full w-full px-0" size="lg">
                <Image
                  src={frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
                  width={1920}
                  height={1080}
                  alt=""
                />
              </Frame>
            </div>
            <div className="mx-6 flex flex-col gap-12 sm:px-4 md:px-12 lg:px-24">
              <MdxContent source={serialized} />
            </div>
          </div>
          <div className="hidden w-2/12 pt-12 text-white xl:ml-6 xl:flex xl:flex-col">
            <p className="text-md text-white/30">Contents</p>
            <div className="relative mt-6 overflow-hidden ">
              {/* <div className="absolute top-0 left-0 z-20 w-full h-full bg-gradient-to-r from-transparent via-[#010101]/30 to-[#010101]/100" /> */}
              {headings.map((heading) => {
                return (
                  <div
                    key={`#${heading.slug}`}
                    className="blog-heading-gradient z-0 my-8 text-ellipsis"
                  >
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
            <div className="flex flex-col">
              <p className="text-md pt-10 text-white/30">Suggested</p>
              <div>
                <SuggestedBlogs currentPostSlug={params.slug} />
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
