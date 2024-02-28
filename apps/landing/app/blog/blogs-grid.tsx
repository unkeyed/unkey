import { authors } from "@/content/blog/authors";
import { Frontmatter, Tags } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { BlogCard } from "./blog-card";
import { BlogPagination } from "./blog-pagination";

type Props = {
  posts: { frontmatter: Frontmatter; slug: string }[];
  className?: string;

  searchParams?: {
    tag?: Tags;
    page?: number;
  };
};

function getAllTags(posts: any[]) {
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

export const BlogGrid: React.FC<Props> = ({ className, posts, searchParams }) => {
  const blogsPerPage: number = 15;
  const allTags = getAllTags(posts);
  const selectedTag = searchParams?.tag;
  const filteredPosts =
    selectedTag && selectedTag !== ("all" as Tags)
      ? posts.filter((p) => p.frontmatter.tags?.includes(selectedTag))
      : posts;

  const page = Number(searchParams?.page ?? 1);
  const visiblePosts = filteredPosts.slice(blogsPerPage * (page - 1), blogsPerPage * page);

  return (
    <div className="">
      <div className={cn("flex flex-wrap py-24 justify-center gap-6 w-full", className)}>
        {allTags.map((tag) => (
          <Link
            key={tag}
            prefetch
            href={tag === "all" ? "/blog" : `/blog?tag=${tag}`}
            className={cn(
              tag === (selectedTag ?? "all") ? "bg-white text-black" : "bg-white/10 text-white/60",
              "py-1 px-3 rounded-lg",
              className,
            )}
          >
            {tag.charAt(0).toUpperCase() + tag.slice(1)}
          </Link>
        ))}
      </div>
      <div className={cn("grid md:grid-cols-2 xl:grid-cols-3 gap-12 mb-24 xxs-mx-auto", className)}>
        {visiblePosts.map((post) => (
          <Link href={`/blog/${post.slug}`} key={post.slug}>
            <BlogCard
              tags={post.frontmatter.tags?.toString()}
              imageUrl={post.frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
              title={post.frontmatter.title}
              subTitle={post.frontmatter.description}
              author={authors[post.frontmatter.author]}
              publishDate={post.frontmatter.date}
            />
          </Link>
        ))}
      </div>
      <div>
        <BlogPagination
          currentPage={page}
          numPages={Math.ceil(filteredPosts.length / blogsPerPage)}
          buildPath={(p: number) => {
            const newParams = new URLSearchParams();
            newParams.set("page", p.toString());
            if (selectedTag) {
              newParams.set("tag", selectedTag);
            }

            // returns this: /blog?page=${p}&tag=${tag}
            return `/blog?${newParams.toString()}`;
          }}
        />
      </div>
    </div>
  );
};
