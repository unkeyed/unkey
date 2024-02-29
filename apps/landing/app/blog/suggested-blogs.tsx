import { BLOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import Link from "next/link";
import React from "react";
import { Frame } from "../../components/frame";

type BlogListProps = {
  className?: string;
  currentPostSlug?: string;
};

export async function SuggestedBlogs({ className, currentPostSlug }: BlogListProps) {
  const posts = (await getAllMDXData({ contentPath: BLOG_PATH }))
    .sort((a, b) => {
      return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
    })
    .filter((p, _i) => p.slug !== currentPostSlug);

  return (
    <div className={cn("flex flex-col w-full mt-8", className)}>
      <Link href={`/blog/${posts[0]?.slug}`} key={posts[0]?.slug}>
        <div className="flex w-full mb-12">
          <div className="flex flex-col gap-4">
            <Frame size="sm">
              <Image
                alt="Blog Image"
                src={`${posts[0]?.frontmatter?.image ?? "/images/blog-images/defaultBlog.png"}`}
                width={600}
                height={400}
              />
            </Frame>
            <p>{posts[0]?.frontmatter?.title}</p>
            <p className="text-white/40 text-sm">
              {format(new Date(posts[0]?.frontmatter.date!), "MMM dd, yyyy")}
            </p>
          </div>
        </div>
      </Link>
      <Link href={`/blog/${posts[1]?.slug}`} key={posts[1]?.slug}>
        <div className="flex w-full mb-12">
          <div className="flex flex-col gap-4">
            <Frame size="sm">
              <Image
                alt="Blog Image"
                src={`${posts[1]?.frontmatter?.image ?? "/images/blog-images/defaultBlog.png"}`}
                width={600}
                height={400}
              />
            </Frame>
            <p>{posts[1]?.frontmatter.title}</p>
            <p className="text-white/40 text-sm">
              {format(new Date(posts[1]?.frontmatter.date!), "MMM dd, yyyy")}
            </p>
          </div>
        </div>
      </Link>
      <Link href={`/blog/${posts[2]?.slug}`} key={posts[2]?.slug}>
        <div className="flex w-full mb-12">
          <div className="flex flex-col gap-4">
            <Frame size="sm">
              <Image
                alt="Blog Image"
                src={`${posts[2]?.frontmatter?.image ?? "/images/blog-images/defaultBlog.png"}`}
                width={600}
                height={400}
              />
            </Frame>
            <p>{posts[2]?.frontmatter.title}</p>
            <p className="text-white/40 text-sm">
              {format(new Date(posts[2]?.frontmatter.date!), "MMM dd, yyyy")}
            </p>
          </div>
        </div>
      </Link>
    </div>
  );
}
