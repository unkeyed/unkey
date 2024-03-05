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
      <BlogContainer className="overflow-hidden mt-32 scroll-smooth ">
        <div>
          <TopLeftShiningLight className="-z-40 h-full" />
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
        <div className="flex xl:flex-row flex-col ">
          <div className="flex flex-col sm:pl-4 lg:pl-24 md:px-12 xl:w-full">
            <h1 className="text-left sm:pt-8 xl:mt-28 text-[40px] sm:text-[56px] text-6xl font-medium tracking-tight blog-heading-gradient leading-[56px] sm:leading-[72px] pr-0 xl:pr-30 xl:w-3/4">
              {frontmatter.title}
            </h1>
            <p className="mt-10 text-lg font-normal leading-8 text-left text-white/40">
              {frontmatter.description}
            </p>
          </div>
          <div className="flex xl:flex-col flex-row  xl:w-80 w-fit pt-12 lg:pl-24 md:px-12 sm:pl-4 xl:pl-12 xl:pt-36">
            <BlogAuthors author={author} className="xl:mb-16 mb-6 w-40" />
            <div className="flex flex-col">
              <p className="mb-6 text-sm text-white/30">Published on</p>
              <h3 className="text-white">{format(new Date(frontmatter.date!), "MMM dd, yyyy")}</h3>
            </div>
          </div>
        </div>

        <div className="flex xl:flex-row flex-col  mb-40">
          <div className="flex flex-col gap-12 w-full">
            <div className="flex ">
              <Frame className="shadow-sm mx-0 px-0 h-full xl:mx-12 w-full" size="lg">
                <Image
                  src={frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
                  width={1200}
                  height={860}
                  alt=""
                />
              </Frame>
            </div>
            <div className="lg:px-24 sm:px-4 md:px-12 flex flex-col gap-12">
              <MdxContent source={serialized} />
            </div>
          </div>
          <div className="pt-12 pl-12 text-white overflow-clip hidden xl:flex xl:flex-col w-80">
            <p className="text-white/30 text-md">Contents</p>
            <div className="relative mt-6 overflow-hidden whitespace-nowrap">
              <div className="absolute top-0 left-0 z-20 w-full h-full bg-gradient-to-r from-transparent via-[#010101]/30 to-[#010101]/100" />
              {headings.map((heading) => {
                return (
                  <div key={`#${heading.slug}`} className="z-0 my-8 text-ellipsis">
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
