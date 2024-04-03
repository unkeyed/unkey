import { authors } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import type { Post } from "contentlayer/generated";
import Link from "next/link";
import { BlogCard } from "./blog-card";
import { BlogPagination } from "./blog-pagination";

type Props = {
  posts: Post[];
  className?: string;

  searchParams?: {
    tag?: string;
    page?: number;
  };
};

function getAllTags(posts: Post[]) {
  const tempTags = ["all"];
  posts.forEach((post) => {
    const newTags = post.tags;
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
    selectedTag && selectedTag !== "all"
      ? posts.filter((p) => p.tags?.includes(selectedTag))
      : posts;

  const page = Number(searchParams?.page ?? 1);
  const visiblePosts = filteredPosts.slice(blogsPerPage * (page - 1), blogsPerPage * page);

  return (
    <div className="">
      <div className={cn("flex flex-wrap py-24 justify-center gap-6 w-full ", className)}>
        {allTags.map((tag) => (
          <Link
            key={tag}
            prefetch
            href={tag === "all" ? "/blog" : `/blog?tag=${tag}`}
            className={cn(
              tag === (selectedTag ?? "all")
                ? "bg-white text-black"
                : "sm:text-sm bg-white/10 text-white/60",
              "py-1 px-3 rounded-lg",
              className,
            )}
          >
            {tag.charAt(0).toUpperCase() + tag.slice(1)}
          </Link>
        ))}
      </div>
      <div className={cn("grid md:grid-cols-2 xl:grid-cols-3 gap-12 mb-24 px-4", className)}>
        {visiblePosts.map((post) => (
          <Link href={`${post._raw.flattenedPath}`} key={post._raw.flattenedPath}>
            <BlogCard
              tags={post.tags}
              imageUrl={post.image ?? "/images/blog-images/defaultBlog.png"}
              title={post.title}
              subTitle={post.description}
              author={authors[post.author]}
              publishDate={post.date}
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
