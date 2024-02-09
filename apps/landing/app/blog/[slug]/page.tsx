import { BlogAuthors } from "@/components/blog/blog-authors";
import { SuggestedBlogs } from "@/components/blog/suggested-blogs";
import { Container } from "@/components/container";
import { CTA } from "@/components/cta";
import { MdxContent } from "@/components/mdx-content";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { BlogBackgroundLines } from "@/components/svg/blog-page";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, getContentData, getFilePaths, getPost } from "@/lib/mdx-helper";
import { format } from "date-fns";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

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
  console.log(JSON.stringify(serialized));

  return (
    <>
      <Container className="scroll-smooth">
        <TopLeftShiningLight />
        <BlogBackgroundLines />
        <TopRightShiningLight />
        <div className="relative mt-16 flex flex-col items-start space-y-8 lg:mt-32 lg:flex-row lg:space-y-0">
          <div className="mx-auto w-full xl:pl-6">
            <h2 className="text-left text-6xl font-medium tracking-tight blog-heading-gradient leading-[72px] pr-0 xl:pr-30 xl:w-3/4">
              {frontmatter.title}
            </h2>
            <p className="mt-10 text-left text-white/40 text-lg font-normal leading-8 xl:pr-40">
              {frontmatter.description}
            </p>
            <div className="bg-black flex flex-col gap-12 pt-16">
              <MdxContent source={serialized} />
            </div>
          </div>

          <div className="top-32 flex h-max w-full flex-col justify-end self-start px-4 sm:px-6 lg:sticky lg:w-2/5 lg:pl-28 gap-8">
            <div>
              <BlogAuthors author={author} className="w-full mb-16" />

              <p className="text-white/30 text-sm mb-6">Published on</p>
              <h3 className="text-white">{format(new Date(frontmatter.date!), "MMM dd, yyyy")}</h3>
            </div>

            <div className="flex text-white w-52 mt-12 overflow-clip max-sm:hidden md:hidden lg:block">
              <p className="text-white/30 text-md">Contents</p>

              <div className="relative mt-6 overflow-hidden whitespace-nowrap">
                <div className="absolute top-0 left-0 w-full h-full bg-gradient-to-r from-transparent via-transparent to-black z-20" />
                {headings.map((heading) => {
                  return (
                    <div key={`#${heading.slug}`} className="my-8 text-ellipsis z-0">
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
                <div className="md:hidden">
                  <p className="text-white/30 text-md pt-10">Suggested</p>
                  <div>
                    <SuggestedBlogs />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <CTA />
      </Container>
    </>
  );
};

export default BlogArticleWrapper;
