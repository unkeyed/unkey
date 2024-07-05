"use client";
import { CopyButton } from "@/components/copy-button";
import { BlogCodeDownload } from "@/components/svg/blog-code-block";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/code-tabs";
import { cn } from "@/lib/utils";
import { Highlight } from "prism-react-renderer";
import { useState } from "react";
import React from "react";
import darkTheme from "./dark-theme";

const CN_BLOG_CODE_BLOCK =
  "flex flex-col bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)] rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)] not-prose text-[0.8125rem]";

export function BlogCodeBlock({ className, children }: any) {
  const blocks = React.Children.map(children, (child: any) => child.props.children.props);

  const buttonLabels = React.Children.map(children, (child: any) =>
    child?.props?.children?.props?.className.replace(/language-/, "").split(" "),
  );
  const [copyData, setCopyData] = useState(blocks[0].children);

  function handleDownload() {
    const element = document.createElement("a");
    const file = new Blob([copyData], { type: "text/plain" });
    element.href = URL.createObjectURL(file);
    element.download = "code.txt";
    document.body.appendChild(element); // Required for this to work in FireFox
    element.click();
  }
  function handlelOnChange(current: any) {
    blocks.map((block: any) => {
      const lang = block.className.replace(/language-/, "");
      if (lang === current[0].toString()) {
        setCopyData(block.children);
      }
    });
  }
  return (
    <div className={cn(CN_BLOG_CODE_BLOCK, className)}>
      <Tabs
        defaultValue={buttonLabels[0]}
        onValueChange={(value) => handlelOnChange(value)}
        className="flex flex-col"
      >
        <div className="flex flex-row border-b-[.5px] border-white/10 p-2">
          <TabsList className="flex flex-row justify-start w-full gap-4 align-bottom h-fit">
            {React.Children.map(buttonLabels, (label: string) => {
              return (
                <TabsTrigger key={label} value={label} className="capitalize text-white/30">
                  {label}
                </TabsTrigger>
              );
            })}
          </TabsList>
          <div className="flex flex-row gap-4 pt-0 pr-4">
            <div>{}</div>
            <CopyButton value={copyData} />
            <button type="button" className="p-0 m-0 bg-transparent" onClick={handleDownload}>
              <BlogCodeDownload />
            </button>
          </div>
        </div>
        {blocks.map((block: any, index: number) => {
          return (
            <TabsContent value={buttonLabels[index]} key={buttonLabels[index]} className="pr-12">
              <Highlight
                theme={darkTheme}
                code={block.children}
                language={block.className.replace(/language-/, "")}
              >
                {({ tokens, getLineProps, getTokenProps }) => (
                  <div className="flex flex-row">
                    <div className="shrink-0 grow-0 flex flex-col py-0 my-0 overflow-x-auto bg-transparent border-none rounded-none">
                      {tokens.map((line, i) => {
                        // if the last line is empty, don't render it
                        if (i === tokens.length - 1 && line[0].empty === true) {
                          return <></>;
                        }
                        return (
                          <pre
                            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                            key={`${line}-${i}`}
                            {...getLineProps({ line })}
                          >
                            <span className="px-3 text-white/20 text-center select-none">
                              {i > 8 ? i + 1 : ` ${i + 1}`}
                            </span>
                          </pre>
                        );
                      })}
                    </div>
                    <div className="flex-grow flex flex-col py-0 my-0 overflow-x-auto bg-transparent border-none rounded-none">
                      {tokens.map((line, i) => {
                        // if the last line is empty, don't render it
                        if (i === tokens.length - 1 && line[0].empty === true) {
                          return <></>;
                        }
                        return (
                          <pre
                            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                            key={`${line}-${i}`}
                            {...getLineProps({ line })}
                          >
                            {line.map((token, key) => (
                              <span key={` ${key}-${token}`} {...getTokenProps({ token })} />
                            ))}
                          </pre>
                        );
                      })}
                    </div>
                  </div>
                )}
              </Highlight>
            </TabsContent>
          );
        })}
      </Tabs>
    </div>
  );
}
export function BlogCodeBlockSingle({ className, children }: any) {
  const block = children.props;

  const [copyData, _setCopyData] = useState(block.children);

  function handleDownload() {
    const element = document.createElement("a");
    const file = new Blob([copyData], { type: "text/plain" });
    element.href = URL.createObjectURL(file);
    element.download = "code.txt";
    document.body.appendChild(element); // Required for this to work in FireFox
    element.click();
  }
  return (
    <div className={cn(CN_BLOG_CODE_BLOCK, className)}>
      <div className="flex flex-row justify-end gap-4 mt-2 mr-4 border-white/10">
        <CopyButton value={copyData} />
        <button
          type="button"
          aria-label="Download code"
          className="p-0 m-0 align-top bg-transparent"
          onClick={handleDownload}
        >
          <BlogCodeDownload />
        </button>
      </div>
      <Highlight
        theme={darkTheme}
        code={block.children}
        language={block.className?.replace(/language-/, "") || "jsx"}
      >
        {({ tokens, getLineProps, getTokenProps }) => {
          return (
            <div className="flex flex-row pt-0 pb-5 mt-0 overflow-x-auto leading-5 bg-transparent border-none rounded-none">
              <div className="shrink-0 grow-0 flex flex-col py-0 my-0 overflow-x-auto bg-transparent border-none rounded-none">
                {tokens.map((line, i) => {
                  // if the last line is empty, don't render it
                  if (i === tokens.length - 1 && line[0].empty === true) {
                    return <></>;
                  }
                  return (
                    <pre
                      // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                      key={`${line}-${i}`}
                      {...getLineProps({ line })}
                    >
                      <span className="px-3 text-white/20 text-center select-none">
                        {i > 8 ? i + 1 : ` ${i + 1}`}
                      </span>
                    </pre>
                  );
                })}
              </div>
              <div className="flex-grow flex flex-col py-0 my-0 overflow-x-auto bg-transparent border-none rounded-none">
                {tokens.map((line, i) => {
                  // if the last line is empty, don't render it
                  if (i === tokens.length - 1 && line[0].empty === true) {
                    return <></>;
                  }
                  return (
                    <pre
                      // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                      key={`${line}-${i}`}
                      {...getLineProps({ line })}
                    >
                      {line.map((token, key) => (
                        <span key={` ${key}-${token}`} {...getTokenProps({ token })} />
                      ))}
                    </pre>
                  );
                })}
              </div>
            </div>
          );
        }}
      </Highlight>
    </div>
  );
}
