import { BlogHero } from "@/components/blog-hero";
import { Container } from "@/components/container";
import { authors } from "@/content/blog/authors";
import { BLOG_PATH, Frontmatter, getAllMDXData } from "@/lib/mdx-helper";

import { BlogGrid } from "@/components/blogs-grid";

// type Props = {
//   params: { slug: string };
//   searchParams: { [key: string]: string | string[] | undefined };
// };

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
export default async function Blog() {
  const posts = (await getAllMDXData({ contentPath: BLOG_PATH })).sort((a, b) => {
    return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
  });
  const _allTags = getAllTags(posts);
  return (
    <>
      <Container className="scroll-smooth">
        <BlogHero
          label={"Product"}
          imageUrl={posts[0].frontmatter.image}
          title={posts[0].frontmatter.title}
          subTitle={posts[0].frontmatter.description}
          author={authors[posts[0].frontmatter.author]}
          publishDate={posts[0].frontmatter.date}
        />
        <BlogGrid posts={posts} />
      </Container>
    </>
  );
}
