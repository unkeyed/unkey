import { CTA } from "@/components/cta";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, type Tags, getAllMDXData } from "@/lib/mdx-helper";
import Link from "next/link";
import { BlogContainer } from "./blog-container";
import { BlogHero } from "./blog-hero";
import { BlogGrid } from "./blogs-grid";

export const metadata = {
  title: "Blog | Unkey",
  description: "Latest blog posts and news from the Unkey team.",
  openGraph: {
    title: "Blog | Unkey",
    description: "Latest blog posts and news from the Unkey team.",
    url: "https://unkey.dev/blog",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Blog | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

type Props = {
  searchParams?: {
    tag?: Tags;
    page?: number;
  };
};

export default async function Blog(props: Props) {
  const posts = (await getAllMDXData({ contentPath: BLOG_PATH })).sort((a, b) => {
    return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
  });
  const postTags: string[] = posts[0].frontmatter.tags?.toString().split(" ") || [];
  return (
    <>
      <BlogContainer className="w-[1440px] mt-32 scroll-smooth">
        <div>
          <TopLeftShiningLight />
        </div>
        <div className="w-full h-full overflow-clip -z-20">
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
            className="hidden overflow-hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={2}
            className="hidden overflow-hidden md:block"
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
            className="hidden overflow-hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={0}
            className="hidden overflow-hidden md:block"
          />
        </div>
        <div>
          <TopRightShiningLight />
        </div>
        <Link href={`/blog/${posts[0].slug}`} key={posts[0].slug}>
          <BlogHero
            tags={postTags}
            imageUrl={posts[0].frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
            title={posts[0].frontmatter.title}
            subTitle={posts[0].frontmatter.description}
            author={authors[posts[0].frontmatter.author]}
            publishDate={posts[0].frontmatter.date}
            className="px-4"
          />
        </Link>
        <BlogGrid posts={posts} searchParams={props.searchParams} />
        <CTA />
      </BlogContainer>
    </>
  );
}
