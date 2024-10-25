"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ChevronUpIcon, ChevronDownIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { useParams } from "next/navigation";
import type { Glossary } from "@/.content-collections/generated";

export default function TermsRolodexDesktop({
  className,
  terms,
}: { className?: string; terms: Array<Pick<Glossary, "slug" | "title">> }) {
  const params = useParams();
  const currentSlug = params.slug;
  if (typeof currentSlug !== "string") {
    throw new Error("slug is not a string");
  }

  const [currentIndex, setCurrentIndex] = useState(() => {
    const initialIndex = terms.findIndex((term) => term.slug === currentSlug);
    return initialIndex >= 0 ? initialIndex : 0;
  });

  const getVisibleTerms = () => {
    const totalVisible = Math.min(7, terms.length);
    const halfVisible = Math.floor(totalVisible / 2);

    const start = currentIndex - halfVisible;
    const end = currentIndex + halfVisible + 1;

    const wrappedTerms = [...terms, ...terms, ...terms];
    const centerOffset = terms.length;

    return wrappedTerms.slice(centerOffset + start, centerOffset + end);
  };

  const handleScroll = (direction: "up" | "down") => {
    setCurrentIndex((prevIndex) => {
      let newIndex = direction === "up" ? prevIndex - 1 : prevIndex + 1;
      if (newIndex < 0) {
        newIndex = terms.length - 1;
      }
      if (newIndex >= terms.length) {
        newIndex = 0;
      }
      return newIndex;
    });
  };

  const visibleTerms = getVisibleTerms();

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex flex-col h-full justify-between content-between space-y-2">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => handleScroll("up")}
          className={cn("p-0 transition-colors duration-150", {
            "cursor-not-allowed": terms.length <= visibleTerms.length,
          })}
          disabled={terms.length <= visibleTerms.length}
        >
          <ChevronUpIcon className="w-4 h-4" />
          <span className="sr-only">Scroll up</span>
        </Button>
        <div className="overflow-hidden flex-grow">
          {visibleTerms.map((term, index) => (
            <Link
              key={`${term.slug}-${index}`}
              href={`/glossary/${term.slug}`}
              className={cn(
                "flex items-center px-2 h-10 rounded-md text-white/60 hover:text-white text-sm font-normal transition-opacity duration-300",
                {
                  "text-white underline":
                    index === Math.floor(visibleTerms.length / 2) && currentSlug !== term.slug,
                  "text-white font-semibold border-l border-white rounded-none":
                    currentSlug === term.slug,
                  "opacity-25":
                    visibleTerms.length > 2 && (index === 0 || index === visibleTerms.length - 1),
                },
              )}
            >
              <span>{term.title}</span>
            </Link>
          ))}
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={() => handleScroll("down")}
          className={cn("p-0 transition-colors duration-150", {
            "cursor-not-allowed": terms.length <= visibleTerms.length,
          })}
          disabled={terms.length <= visibleTerms.length}
        >
          <ChevronDownIcon className="w-4 h-4" />
          <span className="sr-only">Scroll down</span>
        </Button>
      </div>
    </div>
  );
}
