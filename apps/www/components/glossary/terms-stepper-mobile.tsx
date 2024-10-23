"use client";

import type { Glossary } from "@/.content-collections/generated";
import { cn } from "@/lib/utils";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";

export default function TermsStepperMobile({
  className,
  terms,
}: { className?: string; terms: Array<Pick<Glossary, "slug" | "title">> }) {
  const params = useParams();
  const slug = params.slug as string;
  const sortedTerms = terms.sort((a, b) => a.title.localeCompare(b.title));
  const slugIndex = sortedTerms.findIndex((term) => term.slug === slug);
  const startIndex = slugIndex !== -1 ? slugIndex : 0;
  const currentTerm = sortedTerms[startIndex];
  const previousTerm = sortedTerms[(startIndex - 1 + sortedTerms.length) % sortedTerms.length];
  const nextTerm = sortedTerms[(startIndex + 1) % sortedTerms.length];

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex flex-col h-full justify-between content-between">
        <div className="grid grid-cols-3 gap-4 pb-6">
          <Link
            href={`/glossary/${previousTerm.slug}`}
            className="flex w-full flex-row items-center gap-2 rounded-lg p-4 text-sm transition-colors text-white/60 hover:text-white"
          >
            <ChevronLeftIcon className="size-4 shrink-0" />
            {previousTerm.title}
          </Link>

          <div className="text-center py-4">
            <p className="font-medium">{currentTerm.title}</p>
          </div>

          <Link
            href={`/glossary/${nextTerm.slug}`}
            className="flex w-full flex-row items-center gap-2 rounded-lg p-4 text-sm transition-colors text-white/60 hover:text-white"
          >
            {nextTerm.title}
            <ChevronRightIcon className="size-4 shrink-0" />
          </Link>
        </div>
      </div>
    </div>
  );
}
