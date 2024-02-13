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
        <div className="relative flex flex-col items-start mt-16 space-y-8 lg:mt-32 lg:flex-row lg:space-y-0">
          <div className="w-full mx-auto xl:pl-6">
            <h2 className="text-left text-6xl font-medium tracking-tight blog-heading-gradient leading-[72px] pr-0 xl:pr-30 xl:w-3/4">
              {frontmatter.title}
            </h2>
            <p className="mt-10 text-lg font-normal leading-8 text-left text-white/40 xl:pr-40">
              {frontmatter.description}
            </p>
            <div className="flex flex-col gap-12 pt-16 bg-black">
              <MdxContent source={serialized} />
            </div>
          </div>

          <div className="flex flex-col self-start justify-end w-full gap-8 px-4 top-32 h-max sm:px-6 lg:sticky lg:w-2/5 lg:pl-28">
            <div>
              <BlogAuthors author={author} className="w-full mb-16" />

              <p className="mb-6 text-sm text-white/30">Published on</p>
              <h3 className="text-white">{format(new Date(frontmatter.date!), "MMM dd, yyyy")}</h3>
            </div>

            <div className="flex mt-12 text-white w-52 overflow-clip max-sm:hidden md:hidden lg:block">
              <p className="text-white/30 text-md">Contents</p>

              <div className="relative mt-6 overflow-hidden whitespace-nowrap">
                <div className="absolute top-0 left-0 z-20 w-full h-full bg-gradient-to-r from-transparent via-transparent to-black" />
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
                <div className="md:hidden">
                  <p className="pt-10 text-white/30 text-md">Suggested</p>
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
