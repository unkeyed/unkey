"use client";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { Highlight, HighlightProps, themes } from "prism-react-renderer";
import { useState } from "react";
import { BlogCodeCopy, BlogCodeDownload } from "./svg/blog-code-block";

const _theme = {
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
  console.log("Blocks", blocks);
  blocks.map((block: any) => console.log(block.className.replace(/language-/, "")));
  const buttonLabels = children.map((child: any) =>
    child?.props?.children?.props?.className.replace(/language-/, "").split(" "),
  );
  // console.log(blocks[0].className.replace(/language-/, ``));
  // console.log(blocks[1].className.replace(/language-/, ``));
  // console.log(blocks[2].className.replace(/language-/, ``));
  return (
    <div
      className={cn(
        "flex flex-col bg-gradient-to-t from-[rgba(255,255,255,0.1)] to-[rgba(255,255,255,0.07)] rounded-[20px] border-[.5px] border-[rgba(255,255,255,0.1)]",
        className,
      )}
    >
      <Tabs defaultValue={buttonLabels[0]} className="w-[400px]">
        <TabsList>
          {buttonLabels.map((label: string) => {
            return <TabsTrigger value={label}>{label}</TabsTrigger>;
          })}
        </TabsList>
        {blocks.map((block: any) => {
          return (
            <TabsContent value={block.className.replace(/language-/, "").toString()}>
              Hello
            </TabsContent>
          );
        })}
      </Tabs>

      {/* <Tabs
        defaultValue={openTab}
        className="h-full w-full"
        value={openTab}
        onValueChange={setOpenTab}
      >
        <TabsList>
          {blocks.map((block: any) => {
            return (
              <TabsTrigger
                value={block.className.replace(/language-/, ``).split(` `)}
                type="button"
                onClick={() =>
                  handleTabsChange(
                    block.className.replace(/language-/, ``).split(` `)
                  )
                }
                className={cn(
                  block.className.replace(/language-/, ``).split(` `) ===
                    openTab
                    ? "bg-white/10 border-white/10 rounded-lg text-white/60"
                    : "bg-transparent text-white/30",
                  "align-middle px-3 py-2"
                )}
              >
                {block.className.replace(/language-/, ``).split(` `)}
              </TabsTrigger>
            );
          })}
        </TabsList>

        {blocks.map((block: any) => {
          let lang = block.className
            .replace(/language-/, ``)
            .split(` `)
            .toString();
          let code = block.children;
          console.log(code);

          return (
            <div>
              <TabsContent value={lang}>
                <div className="w-[600px] h-full">This is DUMB</div>
              </TabsContent>
            </div>
          );
        })} */}

      {/* <div className="flex py-4 p-4">
          <div className="w-full flex flex-row gap-4 justify-start h-fit align-bottom">
            {buttonLabels.map((label: string) => {
              return (
                <button
                  type="button"
                  onClick={() => handleTabsChange(label)}
                  className={cn(
                    label === current
                      ? "bg-white/10 border-white/10 rounded-lg text-white/60"
                      : "bg-transparent text-white/30",
                    "align-middle px-3 py-2"
                  )}
                >
                  {label}
                </button>
              );
            })}
          </div>
          <div className="flex flex-row gap-4 pr-1 pt-2">
            <BlogCodeCopy />
            <BlogCodeDownload />
          </div>
        </div> */}
      {/* <div className="border-t-[.5px] border-white/10">
          {children.map((child: any) => {
            const { title, ...blockProps } = child.props.children.props;
            return (
              <Highlight
                theme={theme}
                code={blockProps.children}
                language={blockProps.className}
              >
                {({ tokens, getLineProps, getTokenProps }) => (
                  <pre className="leading-10 border-none rounded-none bg-transparent">
                    {tokens.map((line, i) => (
                      <div key={`${line}-${i}`} {...getLineProps({ line })}>
                        <span>{i + 1}</span>
                        {line.map((token, key) => (
                          <span
                            key={` ${key}-${token}`}
                            {...getTokenProps({ token })}
                          />
                        ))}
                      </div>
                    ))}
                  </pre>
                )}
              </Highlight>
            );
          })}
        </div> */}
      {/* </Tabs> */}
    </div>
  );
}
