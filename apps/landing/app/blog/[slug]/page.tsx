import { CTA } from "@/components/cta";
import { Frame } from "@/components/frame";
import { MdxContent } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, getContentData, getFilePaths, getPost } from "@/lib/mdx-helper";
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
      <BlogContainer className="mt-32 overflow-hidden scroll-smooth ">
        <div>
          <TopLeftShiningLight className="hidden h-full -z-40 sm:block" />
        </div>
        <div className="w-full overflow-clip -z-40">
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
        <div className="flex flex-col xl:flex-row">
          <div className="flex flex-col mx-6 sm:pl-4 lg:pl-24 md:px-12 lg:w-10/12 xl:mt-12">
            <h1 className="text-left sm:pt-8 xl:mt-0 text-[40px] sm:text-[56px] text-6xl font-medium tracking-tight blog-heading-gradient leading-[56px] sm:leading-[72px] pr-0 xl:pr-30 xl:w-3/4">
              {frontmatter.title}
            </h1>
            <p className="mt-10 text-lg font-normal leading-8 text-left text-white/40 ">
              {frontmatter.description}
            </p>
          </div>
          <div className="flex flex-col justify-start w-full p-0 pl-6 mt-6 lg:w-2/12">
            <div className="flex flex-row items-start w-full gap-2 mb-8 xl:flex-col lg:gap-12 md:ml-12 lg:ml-24 xl:ml-0">
              <BlogAuthors author={author} className="w-40 mt-0 mb-0 lg:w-full sm:ml-4" />
              <div className="flex flex-col w-full mt-0">
                <p className="mb-0 text-white/30 text-nowrap">Published on</p>
                <p className="pt-1 mt-8 text-white text-nowrap xl:pt-0">
                  {format(new Date(frontmatter.date!), "MMM dd, yyyy")}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div className="flex mb-40 ">
          <div className="flex flex-col w-full gap-12 xl:w-10/12 ">
            <div className="flex ">
              <Frame className="w-full h-full px-0 mx-6 shadow-sm xl:mx-12" size="lg">
                <Image
                  src={frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
                  width={1200}
                  height={860}
                  alt=""
                />
              </Frame>
            </div>
            <div className="flex flex-col gap-12 mx-6 lg:px-24 sm:px-4 md:px-12">
              <MdxContent source={serialized} />
            </div>
          </div>
          <div className="hidden w-2/12 pt-12 text-white xl:flex xl:flex-col xl:ml-6">
            <p className="text-white/30 text-md">Contents</p>
            <div className="relative mt-6 overflow-hidden ">
              {/* <div className="absolute top-0 left-0 z-20 w-full h-full bg-gradient-to-r from-transparent via-[#010101]/30 to-[#010101]/100" /> */}
              {headings.map((heading) => {
                return (
                  <div
                    key={`#${heading.slug}`}
                    className="z-0 my-8 text-ellipsis blog-heading-gradient"
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
              <p className="pt-10 text-white/30 text-md">Suggested</p>
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
