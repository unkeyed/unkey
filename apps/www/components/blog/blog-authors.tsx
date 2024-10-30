"use client";

import type { Author } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { AvatarImage } from "@radix-ui/react-avatar";
import { Avatar, AvatarFallback } from "../../components/ui/avatar";

//Todo: Add ability to have multiple authors

type BlogAuthorsProps = {
  author: Author;
  className?: string;
};
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
export function BlogAuthors({ author, className }: BlogAuthorsProps) {
  return (
    <div className={cn("flex flex-col gap-4 lg:gap-8 xl:gap-16 ", className)}>
      <p className="text-white/40">Written by</p>
      <div className="flex flex-col ">
        <div className="flex items-center gap-2">
          <Avatar className="w-10 h-10">
            <AvatarImage
              alt={author.name}
              src={author.image.src}
              width={12}
              height={12}
              className="w-full"
            />
            <AvatarFallback />
          </Avatar>
          <p className="text-white text-nowrap xl:block">{author.name}</p>
        </div>

        {/* <TooltipProvider>
          <Tooltip>
            <TooltipTrigger className="pt-1 pl-4 xl:hidden">
              <QuestionCircle />
            </TooltipTrigger>
            <TooltipContent
              side="bottom"
              align="center"
              sideOffset={5}
              className="text-white/60"
            >
              <p>{author.name}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider> */}
      </div>
    </div>
  );
}
