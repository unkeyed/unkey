import { CTA } from "@/components/cta";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, Frontmatter, Tags, getAllMDXData } from "@/lib/mdx-helper";
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

function getAllTags(posts: { frontmatter: Frontmatter; slug: string }[]) {
  const tempTags = ["all"];
  posts.forEach((post) => {
    const newTags = post.frontmatter.tags?.toString().split(" ");
    newTags?.forEach((tag: string) => {
      if (!tempTags.includes(tag)) {
        tempTags.push(tag);
      }
    });
  });
  return tempTags;
}

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
  const _allTags = getAllTags(posts);
  const postTags: string[] = posts[0].frontmatter.tags?.toString().split(" ") || [];
  return (
    <>
      <BlogContainer className="max-w-full mt-32 scroll-smooth">
        <TopLeftShiningLight />
        <div className="w-full overflow-clip">
          <MeteorLinesAngular number={2} xPos={0} />
          <MeteorLinesAngular number={2} xPos={200} />
          <MeteorLinesAngular number={2} xPos={400} />
        </div>
        <TopRightShiningLight />
        <Link href={`/blog/${posts[0].slug}`} key={posts[0].slug}>
          <BlogHero
            tags={postTags}
            imageUrl={posts[0].frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
            title={posts[0].frontmatter.title}
            subTitle={posts[0].frontmatter.description}
            author={authors[posts[0].frontmatter.author]}
            publishDate={posts[0].frontmatter.date}
          />
        </Link>
        <BlogGrid posts={posts} searchParams={props.searchParams} />
        <CTA />
      </BlogContainer>
    </>
  );
}
