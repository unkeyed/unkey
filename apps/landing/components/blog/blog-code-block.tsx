"use client";
import { CopyButton } from "@/components/copy-button";
import { BlogCodeDownload } from "@/components/svg/blog-code-block";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/code-tabs";
import { cn } from "@/lib/utils";
import { Highlight } from "prism-react-renderer";
import { useState } from "react";
import React from "react";
import darkTheme from "./dark-theme";

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
    <div
      className={cn(
        "flex flex-col rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)] bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)]",
        className,
      )}
    >
      <Tabs
        defaultValue={buttonLabels[0]}
        onValueChange={(value) => handlelOnChange(value)}
        className="flex flex-col"
      >
        <div className="flex flex-row border-b-[.5px] border-white/10 p-2">
          <TabsList className="flex h-fit w-full flex-row justify-start gap-4 align-bottom">
            {React.Children.map(buttonLabels, (label: string) => {
              return (
                <TabsTrigger key={label} value={label} className="capitalize text-white/30">
                  {label}
                </TabsTrigger>
              );
            })}
          </TabsList>
          <div className="flex flex-row gap-4 pr-4 pt-2">
            <CopyButton value={copyData} />
            <button type="button" className="m-0 bg-transparent p-0" onClick={handleDownload}>
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
                  <pre className="overflow-x-auto rounded-none border-none bg-transparent leading-10 ">
                    {tokens.map((line, i) => (
                      <div
                        // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                        key={`${line}-${i}`}
                        {...getLineProps({ line })}
                      >
                        <span className="pl-4 pr-8 text-center text-white/20">{i + 1}</span>
                        {line.map((token, key) => (
                          <span key={` ${key}-${token}`} {...getTokenProps({ token })} />
                        ))}
                      </div>
                    ))}
                  </pre>
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
    <div
      className={cn(
        "flex flex-col rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)] bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)]",
        className,
      )}
    >
      <div className="flex w-full flex-row justify-end gap-2 border-white/10 p-2 pr-4 ">
        <CopyButton value={copyData} />
        <button type="button" className="m-0 bg-transparent p-0" onClick={handleDownload}>
          <BlogCodeDownload />
        </button>
      </div>
      <Highlight
        theme={darkTheme}
        code={block.children}
        language={block.className.replace(/language-/, "")}
      >
        {({ tokens, getLineProps, getTokenProps }) => (
          <pre className="overflow-x-auto rounded-none border-none bg-transparent leading-10 ">
            {tokens.map((line, i) => (
              <div
                // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                key={`${line}-${i}`}
                {...getLineProps({ line })}
              >
                <span className="pl-4 pr-8 text-center text-white/20">{i + 1}</span>
                {line.map((token, key) => (
                  <span key={` ${key}-${token}`} {...getTokenProps({ token })} />
                ))}
              </div>
            ))}
          </pre>
        )}
      </Highlight>
    </div>
  );
}
