"use client";
import darkTheme from "@/components/blog/darkTheme";
import { CopyButton } from "@/components/copy-button";
import { BlogCodeDownload } from "@/components/svg/blog-code-block";
import { cn } from "@/lib/utils";
import React, { useState } from "react";
import SyntaxHighlighter, { SyntaxHighlighterProps } from "react-syntax-highlighter";

export function CodeBlock(props: any) {
  let language = props.node.children[0].properties?.className;
  // for some reason... occasionally for no reason at all. the className is not in the properties
  if (!language) {
    language = ["language-jsx"];
  }
  const block =
    props.node.children[0].properties?.value || props.node.children[0].children[0].value;
  const [copyData, _setCopyData] = useState(block);

  function handleDownload() {
    const element = document.createElement("a");
    const file = new Blob([copyData], { type: "text/plain" });
    element.href = URL.createObjectURL(file);
    element.download = "code.txt";
    document.body.appendChild(element); // Required for this to work in FireFox
    element.click();
  }
  return (
    <div
      className={cn(
        "flex flex-col bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)] rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)]",
        props.className,
      )}
    >
      <div className="flex flex-row justify-end gap-4 mt-2 mr-4 border-white/10">
        <CopyButton value={copyData} />
        <button
          type="button"
          className="p-0 m-0 align-top bg-transparent"
          onClick={handleDownload}
          aria-label="Download code snippet"
        >
          <BlogCodeDownload />
        </button>
      </div>
      <SyntaxHighlighter
        language={language}
        style={darkTheme}
        showLineNumbers={true}
        wrapLongLines={true}
        wrapLines={true}
        lineProps={(lineNumber) => ({
          style: { display: "block", cursor: "pointer" },
          onClick() {
            alert(`Line Number Clicked: ${lineNumber}`);
          },
        })}
      >
        {block}
      </SyntaxHighlighter>
    </div>
  );
}
