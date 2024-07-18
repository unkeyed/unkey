import { authors } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import type { Post } from "content-collections";
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
    <div>
      <div
        className={cn(
          "flex flex-wrap py-24 justify-center gap-2 sm:gap-4 md:gap-6 w-full",
          className,
        )}
      >
        {allTags.map((tag) => (
          <Link
            scroll={false}
            key={tag}
            prefetch
            href={tag === "all" ? "/blog" : `/blog?tag=${tag}`}
            className={cn(
              tag === (selectedTag ?? "all")
                ? "bg-white text-black"
                : "bg-[rgb(26,26,26)] hover:bg-neutral-800 text-white/60",
              " px-3 rounded-lg h-7 duration-150 ease-out content-center sm:text-sm",
              className,
            )}
          >
            {tag.charAt(0).toUpperCase() + tag.slice(1)}
          </Link>
        ))}
      </div>
      <div
        className={cn(
          "max-sm:flex max-sm:flex-col max-sm:h-full grid md:grid-cols-2 xl:grid-cols-3 gap-12 mb-24 mx-auto",
          className,
        )}
      >
        {visiblePosts.map((post) => (
          <Link href={`${post.url}`} key={`${post.url}`}>
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

            return `/blog?${newParams.toString()}`;
          }}
        />
      </div>
    </div>
  );
};
