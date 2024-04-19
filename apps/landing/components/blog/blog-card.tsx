import type { Author } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import { Frame } from "../frame";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";

type BlogCardProps = {
  tags?: string[];
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
  return (
    <div className={cn("flex flex-col rounded-3xl", className)}>
      <div className="rounded-2xl bg-clip-border w-full">
        <Frame size="sm">
          <div className="relative aspect-video">
            <Image
              src={imageUrl!}
              alt="Hero Image"
              className="object-center overflow-hidden w-full"
              fill={true}
            />
          </div>
        </Frame>
      </div>
      <div className="flex flex-col px-6 h-full ">
        <div className="flex flex-col h-80">
          <div className="flex flex-inline my-4 gap-4">
            {tags?.map((tag) => (
              <div className="text-white/50 text-sm bg-white/10 px-[9px] rounded-md content-center">
                {tag.charAt(0).toUpperCase() + tag.slice(1)}
              </div>
            ))}
            : null
          </div>
          <h2 className="flex justify-start md:text-2xl mt-6 font-medium leading-10 text-xl sm:text-2xl blog-heading-gradient">
            {title}
          </h2>

          <p className="text-base mt-6 font-normal leading-6 sm:text-sm text-white/60 h-full">
            {subTitle}
          </p>
          {/* Todo: Needs ability to add multiple authors at some point */}
          <div className="flex flex-col justify-end h-full flex-wrap">
            <div className="flex flex-row">
              <Avatar className="w-8 h-8">
                <AvatarImage alt={author.name} src={author.image.src} width={12} height={12} />
                <AvatarFallback />
              </Avatar>
              <p className="pt-3 ml-4 text-sm font-medium text-white">{author.name}</p>
              <p className="pt-3 ml-6 text-sm font-normal text-white/40">
                {format(new Date(publishDate!), "MMM dd, yyyy")}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
