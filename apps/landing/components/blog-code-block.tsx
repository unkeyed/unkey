"use client";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/code-tabs";
import { cn } from "@/lib/utils";
import { Highlight } from "prism-react-renderer";
import { useState } from "react";
import { CopyButton } from "./copy-button";
import { BlogCodeDownload } from "./svg/blog-code-block";

const theme = {
  plain: {
    color: "#F8F8F2",
    backgroundColor: "#282A36",
  },
  styles: [
    {
      types: ["string"],
      style: {
        color: "#3CEEAE",
      },
    },
    {
      types: ["string-property"],
      style: {
        color: "#9D72FF",
      },
    },
    {
      types: ["number"],
      style: {
        color: "#FB3186",
      },
    },
  ],
};
export function BlogCodeBlock({ className, children }: any) {
  const blocks = children.map((child: any) => child.props.children.props);

  const buttonLabels = children.map((child: any) =>
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
        "flex flex-col bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)] rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)]",
        className,
      )}
    >
      <Tabs
        defaultValue={buttonLabels[0]}
        onValueChange={(value) => handlelOnChange(value)}
        className="flex flex-col"
      >
        <div className="flex flex-row border-b-[.5px] border-white/10 p-2">
          <TabsList className="w-full flex flex-row gap-4 justify-start h-fit align-bottom">
            {buttonLabels.map((label: string) => {
              return (
                <TabsTrigger value={label} className="capitalize">
                  {label}
                </TabsTrigger>
              );
            })}
          </TabsList>
          <div className="flex flex-row gap-4 pr-4 pt-2">
            <CopyButton value={copyData} />
            <button type="button" className="p-0 m-0 bg-transparent" onClick={handleDownload}>
              <BlogCodeDownload />
            </button>
          </div>
        </div>
        {blocks.map((block: any, index: string) => {
          return (
            <TabsContent value={buttonLabels[index]}>
              <Highlight
                theme={theme}
                code={block.children}
                language={block.className.replace(/language-/, "")}
              >
                {({ tokens, getLineProps, getTokenProps }) => (
                  <pre className="leading-10 border-none rounded-none bg-transparent">
                    {tokens.map((line, i) => (
                      <div key={`${line}-${i}`} {...getLineProps({ line })}>
                        <span className="pl-4 pr-8 text-white/20 text-center">{i + 1}</span>
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
