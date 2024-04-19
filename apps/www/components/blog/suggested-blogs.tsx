import { type Post, allPosts } from "@/.contentlayer/generated";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import Link from "next/link";
import React from "react";
import { Frame } from "../frame";

type BlogListProps = {
  className?: string;
  currentPostSlug?: string;
};

export function SuggestedBlogs({ className, currentPostSlug }: BlogListProps): JSX.Element {
  const posts = allPosts.filter((post: Post, _i) => post.url !== currentPostSlug).slice(0, 3);
  if (posts.length === 0) {
    return <></>;
  }
  return (
    <div>
      {posts.map((post) => (
        <div className={cn("flex flex-col w-full mt-8 prose", className)}>
          <Link href={post.url} key={post.url}>
            <div className="flex w-full">
              <div className="flex flex-col gap-2">
                <Frame size="sm">
                  <Image
                    alt="Blog Image"
                    src={post.image ?? "/images/blog-images/defaultBlog.png"}
                    width={600}
                    height={400}
                  />
                </Frame>
                <p className="text-white">{post?.title}</p>
                <p className="text-sm text-white/50">
                  {format(new Date(post?.date!), "MMM dd, yyyy")}
                </p>
              </div>
            </div>
          </Link>
        </div>
      ))}
    </div>
  );
}
