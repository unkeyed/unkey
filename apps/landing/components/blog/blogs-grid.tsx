"use client";
import { authors } from "@/content/blog/authors";
import { Frontmatter } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useEffect, useState } from "react";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "../ui/pagination";
import { BlogCard } from "./blog-card";

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
  const blogsPerPage: number = 15;
  const [activeTag, setActiveTag] = useState("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [filteredPosts, setFilteredPosts] = useState(posts.slice(0, blogsPerPage));
  const [currentPageCount, setCurrentPageCount] = useState(Math.ceil(posts.length / blogsPerPage));
  const allTags = getAllTags(posts);

  useEffect(() => {
    updatePosts(activeTag);
  }, [currentPage]);

  function updatePage(page: number) {
    if (page < 1 || page > currentPageCount) {
      return;
    }
    setCurrentPage(page);
    console.log("Current Page is ", page);
  }
  function updatePosts(tag: string) {
    setActiveTag(tag);
    //console.log("Active Tag is ", tag);
    let updatedPageCount = 0;
    const sliceStart = (currentPage - 1) * blogsPerPage;
    const sliceEnd = currentPage * blogsPerPage;
    let currentPosts: { frontmatter: Frontmatter; slug: string }[] = [];
    if (tag === "all") {
      currentPosts = posts.slice(sliceStart, sliceEnd);
      setFilteredPosts(currentPosts);
      updatedPageCount = Math.ceil(posts.length / blogsPerPage);
      setCurrentPageCount(updatedPageCount);
      return;
    }
    posts.filter((post) => {
      if (post.frontmatter.tags?.toString().includes(tag)) {
        currentPosts = [...currentPosts, post];
      }
    });
    updatedPageCount = Math.ceil(currentPosts.length / blogsPerPage);
    setCurrentPageCount(updatedPageCount);
    console.log("Current Posts Length before slice", currentPosts.length);
    if (currentPosts.length > blogsPerPage) {
      currentPosts = currentPosts.slice(sliceStart, sliceEnd);
    }
    console.log("Current Posts Length after slice", currentPosts.length);
    setFilteredPosts(currentPosts);
    return;
  }
  function GetPageButtons() {
    const content = [];
    for (let count = 1; count <= currentPageCount; count++) {
      const isEllipses =
        (count > currentPage + 2 && count === currentPageCount) ||
        (count <= currentPage - 2 && count === 2);

      if (!isEllipses) {
        content.push(
          <PaginationLink
            isActive={currentPage === count ? true : false}
            onClick={() => updatePage(count)}
          >
            {count}
          </PaginationLink>,
        );
      } else {
        content.push(<PaginationEllipsis />);
      }
    }
    return content;
  }
  function SetupPagination() {
    if (currentPageCount <= 1) {
      return;
    }
    return (
      <Pagination>
        <PaginationContent>
          <PaginationItem>
            <PaginationPrevious onClick={() => updatePage(currentPage - 1)} />
          </PaginationItem>
          <GetPageButtons />
          <PaginationItem>
            <PaginationNext onClick={() => updatePage(currentPage + 1)} />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    );
  }

  return (
    <div>
      <div className={cn("flex flex-wrap py-24 justify-center gap-6 w-full", className)}>
        {allTags.map((tag) => (
          <button
            type="button"
            onClick={() => updatePosts(tag)}
            className={cn(
              activeTag === tag ? "bg-white text-black" : "bg-white/10 text-white/60",
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
      <SetupPagination />
    </div>
  );
};
