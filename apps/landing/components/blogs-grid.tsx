"use client";
import { authors } from "@/content/blog/authors";
import { Frontmatter } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import { useEffect, useState } from "react";
import { BlogCard } from "./blog-card";
import {
  Pagination,
  PaginationContent,
  // PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "./ui/pagination";

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
  const blogsPerPage: number = 6;
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
    console.log("Current Page is ", currentPage);
  }
  function updatePosts(tag: string) {
    setActiveTag(tag);
    //console.log("Active Tag is ", tag);
    let updatedPageCount = 0;
    const sliceStart = (currentPage - 1) * blogsPerPage;
    const sliceEnd = currentPage * blogsPerPage;
    let currentPosts: { frontmatter: Frontmatter; slug: string }[] = [];
    if (tag === "all") {
      //console.log(sliceStart, sliceEnd);

      currentPosts = posts.slice(sliceStart, sliceEnd);
      setFilteredPosts(currentPosts);
      updatedPageCount = Math.ceil(posts.length / blogsPerPage);
      setCurrentPageCount(updatedPageCount);
      //console.log("Current Page Count is ", currentPageCount);
      return;
    }
    posts.filter((post) => {
      if (post.frontmatter.tags?.toString().includes(tag)) {
        currentPosts = [...currentPosts, post];
        updatedPageCount = Math.ceil(currentPosts.length / blogsPerPage);
        setCurrentPageCount(updatedPageCount);
        if (currentPosts.length > blogsPerPage) {
          currentPosts = currentPosts.slice(sliceStart, sliceEnd);
        }

        setFilteredPosts(currentPosts);
        return;
      }
    });

    //console.log("Current Posts Length", currentPosts.length);

    return;
  }
  function GetPageButtons() {
    const content = [];
    for (let count = 1; count <= currentPageCount; count++) {
      content.push(
        <PaginationLink
          isActive={currentPage === count ? true : false}
          onClick={() => updatePage(count)}
        >
          {count}
        </PaginationLink>,
      );
    }
    return content;
  }
  function SetupPagination() {
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
      <div className={cn("flex flex-row py-24 w-full justify-center gap-6", className)}>
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
      <div className={cn("grid grid-cols-3 gap-12", className)}>
        {filteredPosts.map((post) => (
          <BlogCard
            tags={post.frontmatter.tags?.toString()}
            imageUrl={post.frontmatter.image}
            title={post.frontmatter.title}
            subTitle={post.frontmatter.description}
            author={authors[post.frontmatter.author]}
            publishDate={post.frontmatter.date}
          />
        ))}
      </div>
      <SetupPagination />
    </div>
  );
};
