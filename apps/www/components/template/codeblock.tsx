"use client";
import { CopyButton } from "@/components/copy-button";
import { BlogCodeDownload } from "@/components/svg/blog-code-block";
import darkTheme from "@/components/template/darkTheme";
import { cn } from "@/lib/utils";
import { Highlight } from "prism-react-renderer";
import { useState } from "react";

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
      <Highlight theme={darkTheme} code={block} language={language[0].replace(/language-/, "")}>
        {({ tokens, getLineProps, getTokenProps }) => {
          return (
            <pre className="pt-0 pb-5 mt-0 overflow-x-auto leading-7 bg-transparent border-none rounded-none">
              {tokens.map((line, i) => {
                // if the last line is empty, don't render it
                if (i === tokens.length - 1 && line[0].empty === true) {
                  return null;
                }
                return (
                  <div
                    // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                    key={`${line}-${i}`}
                    {...getLineProps({ line })}
                  >
                    <span className="pl-4 pr-8 text-center text-white/20">{i + 1}</span>
                    {line.map((token, key) => (
                      <span key={` ${key}-${token}`} {...getTokenProps({ token })} />
                    ))}
                    <span className="pl-6"></span>
                  </div>
                );
              })}
            </pre>
          );
        }}
      </Highlight>
    </div>
  );
}
