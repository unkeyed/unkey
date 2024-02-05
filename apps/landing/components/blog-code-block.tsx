"use client";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { BlogCodeCopy, BlogCodeDownload } from "./svg/blog-code-block";

const languages = ["npm", "javascript", "typescript", "go", "rust", "json"];

type CodeBlockProps = {
  children?: React.ReactNode;
  className?: string;
  codeType?: string;
};

export function BlogCodeBlock({ children, className, ...props }: CodeBlockProps) {
  const [current, setCurrent] = useState(props.codeType || "typescript");
  return (
    <div
      className={cn(
        "flex flex-col bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)] rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)]",
        className,
      )}
    >
      <div className="flex py-4 p-4">
        <div className="w-full flex flex-row gap-4 justify-start h-fit align-bottom">
          {languages.map((lang) => {
            return (
              <button
                type="button"
                onClick={() => setCurrent(lang)}
                className={cn(
                  lang === current
                    ? "bg-white/10 border-white/10 rounded-lg text-white/60"
                    : "bg-transparent text-white/30",
                  "align-middle px-3 py-2",
                )}
              >
                {lang}
              </button>
            );
          })}
        </div>
        <div className="flex flex-row gap-4 pr-1 pt-2">
          <BlogCodeCopy />
          <BlogCodeDownload />
        </div>
      </div>
      <pre>
        <code className="w-full h-96 bg-transparent text-blue-500 p-4 rounded-b-[20px] border-t-[.5px] border-[rgba(255,255,255,0.1)] resize-none">
          {children}
        </code>
      </pre>
    </div>
  );
}
