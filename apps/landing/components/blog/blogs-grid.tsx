"use client";
import { authors } from "@/content/blog/authors";
import { Frontmatter } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { BlogCard } from "./blog-card";
import { BlogPagination } from "./blog-pagination";

type Props = {
  posts: any[];
  className?: string;
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

export const BlogGrid: React.FC<Props> = ({ className, posts }) => {
  const searchParams = useSearchParams();
  const pathname = usePathname();
  const { replace } = useRouter();
  const params = new URLSearchParams(searchParams);
  const blogsPerPage: number = 15;
  const [filteredPosts, setFilteredPosts] = useState(posts.slice(0, blogsPerPage));
  const [currentPageCount, setCurrentPageCount] = useState(Math.ceil(posts.length / blogsPerPage));
  const allTags = getAllTags(posts);

  useEffect(() => {
    updatePosts();
  }, [params.get("page"), params.get("tag")]);

  function updateTag(tag: string) {
    params.set("tag", tag);
    replace(`${pathname}?${params.toString()}`);
  }

  function updatePageNumber(page: number) {
    params.set("page", page.toString());
    replace(`${pathname}?${params.toString()}`);
  }

  function updatePosts() {
    const currentPage = params.get("page") ? parseInt(params.get("page") as string) : 1;
    const currentTag = params.get("tag");
    let updatedPageCount = 0;
    const sliceStart = (currentPage - 1) * blogsPerPage;
    const sliceEnd = currentPage * blogsPerPage;
    let currentPosts: { frontmatter: Frontmatter; slug: string }[] = [];
    if (currentTag === "all") {
      currentPosts = posts.slice(sliceStart, sliceEnd);
      setFilteredPosts(currentPosts);
      updatedPageCount = Math.ceil(posts.length / blogsPerPage);
      setCurrentPageCount(updatedPageCount);
      return;
    }
    posts.filter((post) => {
      if (post.frontmatter.tags?.toString().includes(currentTag)) {
        currentPosts = [...currentPosts, post];
      }
    });
    if (currentPosts.length >= blogsPerPage) {
      updatedPageCount = Math.ceil(currentPosts.length / blogsPerPage);
      setCurrentPageCount(updatedPageCount);
      currentPosts = posts.slice(sliceStart, sliceEnd);
      setFilteredPosts(currentPosts);
    }
    updatedPageCount = 1;
    setCurrentPageCount(updatedPageCount);
    setFilteredPosts(currentPosts);
    return;
  }

  return (
    <div>
      <div className={cn("flex flex-wrap py-24 justify-center gap-6 w-full", className)}>
        {allTags.map((tag) => (
          <button
            type="button"
            key={tag}
            onClick={() => updateTag(tag)}
            className={cn(
              (!params.get("page") && tag === "all") || tag === params.get("tag")
                ? "bg-white text-black"
                : "bg-white/10 text-white/60",
              "py-1 px-3 rounded-lg",
              className,
            )}
          >
            {tag.charAt(0).toUpperCase() + tag.slice(1)}
          </button>
        ))}
      </div>
      <div className={cn("grid md:grid-cols-2 xl:grid-cols-3 gap-12 mb-24", className)}>
        {filteredPosts.map((post) => (
          <Link href={`/blog/${post.slug}`} key={post.slug}>
            <BlogCard
              tags={post.frontmatter.tags?.toString()}
              imageUrl={post.frontmatter.image}
              title={post.frontmatter.title}
              subTitle={post.frontmatter.description}
              author={authors[post.frontmatter.author]}
              publishDate={post.frontmatter.date}
            />
          </Link>
        ))}
      </div>
      <BlogPagination
        currentPage={parseInt(params.get("page") as string)}
        numPages={currentPageCount}
        updatePageNumber={updatePageNumber}
      />
    </div>
  );
};
