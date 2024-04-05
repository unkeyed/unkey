import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import type { Author } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import { Frame } from "../frame";
export function QuestionCircle({ className }: { className?: string }) {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="12" cy="12" r="7.5" stroke="white" />
      <path
        d="M9.98389 9C10.17 8.62441 10.4574 8.30833 10.8136 8.08745C11.1698 7.86657 11.5807 7.74969 11.9999 7.75C12.4464 7.74998 12.8827 7.88278 13.2535 8.13152C13.6243 8.38025 13.9126 8.73367 14.082 9.14679C14.2513 9.55992 14.2938 10.0141 14.2042 10.4515C14.1147 10.8889 13.897 11.2897 13.5789 11.603C13.0789 12.096 12.4709 12.628 12.1769 13.253M11.9999 16.25V16.26"
        stroke="white"
        strokeLinecap="round"
        strokeLinejoin="round"
        className={className}
      />
    </svg>
  );
}

type BlogHeroProps = {
  tags?: string[];
  imageUrl?: string;
  title?: string;
  subTitle?: string;
  author: Author;
  publishDate?: string;
  className?: string;
};

export function BlogHero({
  tags,
  imageUrl,
  title,
  subTitle,
  author,
  publishDate,
  className,
}: BlogHeroProps) {
  return (
    <div
      className={cn("flex flex-col lg:flex-row w-full gap-8 xl:gap-16 relative z-100", className)}
    >
      <div className="flex flex-col h-full w-full  lg:w-1/2">
        <Frame className="order-2 w-full lg:order-1 z-100 h-full" size="lg">
          <Image src={imageUrl!} width={1920} height={1080} alt="Hero Image" />
        </Frame>
      </div>
      <div className="lg:w-1/2">
        <div className="flex flex-col order-1 lg:order-2 z-100 h-full justify-evenly">
          <div className="flex flex-row gap-4 lg:justify-start justify-center pb-4">
            {tags?.map((tag) => (
              <p
                key={tag}
                className="text-white/50 text-sm bg-[rgb(26,26,26)] px-[9px] rounded-md w-fit leading-6 flex items-center capitalize py-.5 z-100"
              >
                {tag.charAt(0).toUpperCase() + tag.slice(1)}
              </p>
            ))}
          </div>
          <h2 className="flex justify-center text-3xl font-medium leading-10 blog-heading-gradient text-center lg:justify-start lg:text-left w-full">
            {title}
          </h2>
          <p className="flex text-center lg:text-left justify-center text-base font-normal leading-7 text-white/60 lg:justify-start">
            {subTitle}
          </p>
          <div className="flex flex-row justify-center w-full lg:justify-start gap-6 pt-6">
            <div className="flex flex-col text-nowrap gap-2">
              <span className="text-sm text-white/30 ">Written by</span>
              <div className="flex items-center gap-4">
                {/* Todo: Needs ability to add multiple authors at some point */}

                <Avatar className="mt-4">
                  <AvatarImage alt={author.name} src={author.image.src} width={12} height={12} />
                  <AvatarFallback />
                </Avatar>
                <span className="text-sm text-white mt-4">{author.name}</span>
                <div />
              </div>
            </div>
            <div className="flex flex-col gap-8">
              <span className="text-sm text-white/30">Published on</span>
              <div>
                <span className="text-sm text-white pt-2">
                  {format(new Date(publishDate!), "MMM dd, yyyy")}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
