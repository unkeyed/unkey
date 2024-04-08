"use client";
import darkTheme from "@/components/blog/dark-theme";
import { CopyButton } from "@/components/copy-button";
import { BlogCodeDownload } from "@/components/svg/blog-code-block";
import { cn } from "@/lib/utils";
import { Highlight } from "prism-react-renderer";
import React, { useState } from "react";

export function CodeBlock(props: any) {
  console.log(props.node.children[0].properties);
  let language = props.node.children[0].properties?.className;
  if (!language) {
    language = "language-jsx";
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
      <div className="flex flex-row gap-4 border-white/10 justify-end mt-2 mr-4">
        <CopyButton value={copyData} />
        <button type="button" className="p-0 m-0 bg-transparent align-top" onClick={handleDownload}>
          <BlogCodeDownload />
        </button>
      </div>
      <Highlight theme={darkTheme} code={block} language={language[0].replace(/language-/, "")}>
        {({ tokens, getLineProps, getTokenProps }) => {
          if (tokens.length > 1) {
            tokens.pop();
          }
          return (
            <pre className="leading-7 border-none rounded-none bg-transparent overflow-x-auto pb-5 pt-0 mt-0">
              {tokens.map((line, i) => (
                <div
                  // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                  key={`${line}-${i}`}
                  {...getLineProps({ line })}
                >
                  <span className="pl-4 pr-8 text-white/20 text-center">{i + 1}</span>
                  {line.map((token, key) => (
                    <span key={` ${key}-${token}`} {...getTokenProps({ token })} />
                  ))}
                </div>
              ))}
            </pre>
          );
        }}
      </Highlight>
    </div>
  );
}
