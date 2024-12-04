"use client";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { ChevronRight } from "lucide-react";
import { type PropsWithChildren, useState } from "react";
import reactElementToJSXString from "react-element-to-jsx-string";

type Props = {
  customCodeSnippet?: string;
};

export const RenderComponentWithSnippet: React.FC<PropsWithChildren<Props>> = (props) => {
  const [open, setOpen] = useState(false);
  const snippet =
    props.customCodeSnippet ??
    reactElementToJSXString(props.children, {
      showFunctions: true,
      useBooleanShorthandSyntax: true,

      displayName: (node) => {
        // @ts-ignore
        return node?.type.displayName ?? "Unknown";
      },
    });
  return (
    <div className="rounded-lg border border-gray-6 bg-gray-1 overflow-hidden">
      <div className="p-8 xl:p-12">{props.children}</div>

      <div className="bg-gray-3 p-2 flex items-center justify-start  border-t border-b border-gray-6">
        <Button variant="ghost" onClick={() => setOpen(!open)}>
          <ChevronRight
            className={cn("transition-all", {
              "rotate-90": open,
            })}
          />{" "}
          Code
        </Button>
      </div>

      <div
        className={cn("w-full  bg-gray-2 transition-all max-h-96 overflow-y-scroll", {
          hidden: !open,
        })}
      >
        <div className="flex items-start">
          <pre className="py-0 text-gray-8 text-right">
            {snippet
              .split("\n")
              .map((_line, i) => i + 1)
              .join("\n")}
          </pre>
          <pre className="py-0 text-gray-12 w-full">{snippet}</pre>
        </div>
      </div>
    </div>
  );
};
