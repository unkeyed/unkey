"use client";

import { Author } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import { Frame } from "../frame";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";

type BlogCardProps = {
  tags?: string;
  imageUrl?: string;
  title?: string;
  subTitle?: string;
  author: Author;
  publishDate?: string;
  className?: string;
};

export function BlogCard({
  tags,
  imageUrl,
  title,
  subTitle,
  author,
  publishDate,
  className,
}: BlogCardProps) {
  const tagList = tags?.split(" ") || [];
  return (
    <div className={cn("flex flex-col p-0 m-0 gap-4", className)}>
      <div className="rounded-2xl bg-clip-border overflow-clip">
        <Frame size="sm" className="max-sm:w-4/5">
          <Image src={imageUrl!} width={1920} height={1080} alt="Hero Image" />
        </Frame>
      </div>
      <div className="flex flex-row gap-3">
        {tagList?.map((tag) => (
          <div className="text-white/50 text-sm bg-white/10 px-[9px] rounded-md w-fit leading-6 my-4">
            {tag.charAt(0).toUpperCase() + tag.slice(1)}
          </div>
        ))}{" "}
        : null
      </div>

      <h2 className="font-medium text-3xl leading-10 blog-heading-gradient">{title}</h2>
      <p className="text-base leading-6 font-normal text-white/60 line-clamp-2">{subTitle}</p>
      <div className="flex flex-row">
        {/* Todo: Needs ability to add multiple authors at some point */}
        <Avatar className="w-8 h-8 mt-2">
          <AvatarImage alt={author.name} src={author.image.src} width={12} height={12} />
          <AvatarFallback />
        </Avatar>
        <p className="text-white pt-3 text-sm font-medium ml-4">{author.name}</p>
        <p className="text-white/40 text-sm pt-3 ml-6 font-normal">
          {format(new Date(publishDate!), "MMM dd, yyyy")}
        </p>
        <div />
      </div>
    </div>
  );
}
