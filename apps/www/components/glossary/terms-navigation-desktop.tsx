"use client";
import { terms } from "@/app/glossary/data";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";

export default function TermsNavigationDesktop({ className }: { className?: string }) {
  const params = useParams();
  const slug = params.slug as string;
  const sortedTerms = terms.sort((a, b) => a.title.localeCompare(b.title));
  const [startIndex, setStartIndex] = useState(() => {
    const slugIndex = sortedTerms.findIndex((term) => term.slug === slug);
    return slugIndex !== -1 ? slugIndex : 0;
  });
  const visibleTerms = Array.from({ length: 7 }, (_, i) => {
    // we want to show the 3 terms before and after the current term, plus the current term
    const index = (startIndex - 3 + i + sortedTerms.length) % sortedTerms.length;
    return sortedTerms[index];
  });

  const [_, setIsScrollingUp] = useState(false);
  const [__, setIsScrollingDown] = useState(false);
  const scrollIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const scroll = useCallback(
    (direction: "up" | "down") => {
      setStartIndex((prevIndex) => {
        if (direction === "up") {
          return (prevIndex - 1 + sortedTerms.length) % sortedTerms.length;
        }
        return (prevIndex + 1) % sortedTerms.length;
      });
    },
    [sortedTerms.length],
  );

  const startScrolling = useCallback(
    (direction: "up" | "down") => {
      if (direction === "up") {
        setIsScrollingUp(true);
      } else {
        setIsScrollingDown(true);
      }
      scroll(direction);
      scrollIntervalRef.current = setInterval(() => scroll(direction), 150);
    },
    [scroll],
  );

  const stopScrolling = useCallback(() => {
    if (scrollIntervalRef.current) {
      clearInterval(scrollIntervalRef.current);
      scrollIntervalRef.current = null;
    }
    setIsScrollingUp(false);
    setIsScrollingDown(false);
  }, []);

  useEffect(() => {
    return () => {
      if (scrollIntervalRef.current) {
        clearInterval(scrollIntervalRef.current);
      }
    };
  }, []);

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex flex-col h-full justify-between content-between">
        <Button
          variant="ghost"
          size="icon"
          onMouseDown={() => startScrolling("up")}
          onMouseUp={stopScrolling}
          onMouseLeave={stopScrolling}
          className={cn("p-0 transition-colors duration-150")}
        >
          <ChevronUpIcon className="w-4 h-4" />
          <span className="sr-only">Scroll up</span>
        </Button>
        <div className="overflow-hidden flex-grow">
          {visibleTerms.map((term) => (
            <Link
              href={`/glossary/${term.slug}`}
              className={cn(
                "flex items-center px-2 h-10 rounded-md text-white/60 hover:text-white",
                {
                  "text-white": slug === term.slug,
                },
              )}
            >
              <span className="text-sm font-normal">{term.title}</span>
            </Link>
          ))}
        </div>
        <Button
          variant="ghost"
          size="icon"
          onMouseDown={() => startScrolling("down")}
          onMouseUp={stopScrolling}
          onMouseLeave={stopScrolling}
          className="p-0 transition-colors duration-150"
        >
          <ChevronDownIcon className="w-4 h-4" />
          <span className="sr-only">Scroll down</span>
        </Button>
      </div>
    </div>
  );
}
