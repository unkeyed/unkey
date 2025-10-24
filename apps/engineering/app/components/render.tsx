"use client";
import { ChevronRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
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
        try {
          return getComponentDisplayName(node);
        } catch (error) {
          console.warn("Failed to get display name:", error);
          return "Unknown";
        }
      },
      functionValue: (fn) => {
        return fn.name || "anonymous";
      },
      filterProps: (_, key) => {
        // Filter out internal React props
        return !key.startsWith("_") && !key.startsWith("$");
      },
    });

  return (
    <div className="rounded-lg border border-gray-6 bg-gray-1 overflow-hidden">
      <div className="p-8 xl:p-12">{props.children}</div>
      <div className="bg-gray-3 p-2 flex items-center justify-start border-t border-b border-gray-6">
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
        className={cn("w-full bg-gray-2 transition-all max-h-96 overflow-y-scroll", {
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

// biome-ignore lint/suspicious/noExplicitAny: Safe to leave
const getComponentDisplayName = (element: any): string => {
  // biome-ignore lint/style/useBlockStatements: <explanation>
  if (!element) return "Unknown";

  // Check for displayName
  if (element.type?.displayName) {
    return element.type.displayName;
  }

  // Check for name property
  if (element.type?.name) {
    return element.type.name;
  }

  // Check if it's a string (HTML element)
  if (typeof element.type === "string") {
    return element.type;
  }

  // Check function name for functional components
  if (typeof element.type === "function") {
    return element.type.name || "AnonymousComponent";
  }

  // Fallback
  return "Unknown";
};
