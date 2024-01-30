"use client";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Author } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import { Frame } from "./frame";
import { Avatar, AvatarFallback, AvatarImage } from "./ui/avatar";

export function QuestionCircle({ className }: { className?: string }) {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="12" cy="12" r="7.5" stroke="white" />
      <path
        d="M9.98389 9C10.17 8.62441 10.4574 8.30833 10.8136 8.08745C11.1698 7.86657 11.5807 7.74969 11.9999 7.75C12.4464 7.74998 12.8827 7.88278 13.2535 8.13152C13.6243 8.38025 13.9126 8.73367 14.082 9.14679C14.2513 9.55992 14.2938 10.0141 14.2042 10.4515C14.1147 10.8889 13.897 11.2897 13.5789 11.603C13.0789 12.096 12.4709 12.628 12.1769 13.253M11.9999 16.25V16.26"
        stroke="white"
        stroke-linecap="round"
        stroke-linejoin="round"
        className={className}
      />
    </svg>
  );
}

type BlogHeroProps = {
  label?: string;
  imageUrl?: string;
  title?: string;
  subTitle?: string;
  author: Author;
  publishDate?: string;
  className?: string;
};

export function BlogHero({
  label,
  imageUrl,
  title,
  subTitle,
  author,
  publishDate,
  className,
}: BlogHeroProps) {
  return (
    <div className={cn("flex flex-col lg:flex-row w-full", className)}>
      <Frame className="h-fit my-auto shadow-sm">
        <Image src={imageUrl!} width={1920} height={1080} alt="Hero Image" />
      </Frame>
      <div className="w-full p-16">
        <div className="relative top-0 left-0 text-white/50 text-sm bg-white/10 px-[9px] rounded-md w-fit leading-6 ">
          {label}
        </div>
        <h2 className="font-medium text-3xl leading-10 blog-heading-gradient my-6">{title}</h2>
        <p className="text-base leading-6 font-normal text-white/60">{subTitle}</p>
        <div className="flex flex-row w-full mt-10 gap-24">
          <div className="flex flex-col gap-6 text-nowrap">
            <p className="text-white/30 text-sm ">Written by</p>
            <div className="flex flex-row">
              {/* Todo: Needs ability to add multiple authors at some point */}
              <Avatar>
                <AvatarImage alt={author.name} src={author.image.src} width={12} height={12} />
                <AvatarFallback />
              </Avatar>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger className="pt-1 pl-4">
                    <QuestionCircle />
                  </TooltipTrigger>
                  <TooltipContent side="bottom" align="center" sideOffset={5}>
                    <p>{author.name}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <div />
            </div>
          </div>
          <div className="flex flex-col gap-6">
            <p className="text-white/30 text-sm">Published on</p>
            <div>
              <p className="text-white text-sm pt-3">
                {format(new Date(publishDate!), "MMM dd, yyyy")}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
