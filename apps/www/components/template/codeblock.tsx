"use client";
import darkTheme from "@/components/blog/darkTheme";
import { CopyButton } from "@/components/copy-button";
import { BlogCodeDownload } from "@/components/svg/blog-code-block";
import { cn } from "@/lib/utils";
import React, { useState } from "react";
import SyntaxHighlighter from "react-syntax-highlighter";

export function CodeBlock(props: any) {
  let language = props.node.children[0].properties?.className;
  // for some reason... occasionally for no reason at all. the className is not in the properties
  if (!language) {
    language = ["language-jsx"];
  }

  const block =
    props.node.children[0].properties?.value ||
    props.node.children[0].children[0].value.trim().replace(/\s+/g, "\n");

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
        "flex flex-col bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)] rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)] my-4 w-full mx-0",
        props.className,
      )}
    >
      <div className="flex flex-row justify-end gap-4 mt-2 border-white/10 pr-3">
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
        language={language.toString().replace(/language-/, "")}
        style={darkTheme}
        showLineNumbers={true}
        wrapLongLines={false}
        customStyle={{ margin: 0, padding: "1rem" }}
        {...props}
        codeTagProps={{
          style: { backgroundColor: "transparent", margin: 0, padding: 0 },
        }}
      >
        {block}
      </SyntaxHighlighter>
    </div>
  );
}
