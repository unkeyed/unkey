import { BLOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import React from "react";
import { Frame } from "./frame";

type BlogListProps = {
  className?: string;
};

export async function SuggestedBlogs({ className }: BlogListProps) {
  const posts = (await getAllMDXData({ contentPath: BLOG_PATH }))
    .sort((a, b) => {
      return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
    })
    .filter((_, i) => i < 3);

  return (
    <div className={cn("flex flex-col w-full mt-8", className)}>
      {posts.map((post) => {
        return (
          <div className="flex  w-full mb-12">
            <div className="flex flex-col gap-4">
              <Frame size="sm">
                <Image
                  alt="Blog Image"
                  src={`${post?.frontmatter?.image}`}
                  width={600}
                  height={400}
                />
              </Frame>
              <p>{post.frontmatter.title}</p>
              <p className="text-white/40 text-sm">
                {format(new Date(post.frontmatter.date!), "MMM dd, yyyy")}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
