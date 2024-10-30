"use client";
import { cn } from "@/lib/utils";
import { useState } from "react";

type Props = {
  tags: string[];
  className?: string;
};

export const TagFilter: React.FC<Props> = ({ className, tags }) => {
  const [activeTag, setActiveTag] = useState("all");
  function filterPosts(tag: string) {
    setActiveTag(tag);
  }
  return (
    <div className={cn("flex flex-row py-24 w-full justify-center gap-6", className)}>
      {tags.map((tag) => (
        <button
          type="button"
          onClick={() => filterPosts(tag)}
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
  );
};
