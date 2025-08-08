"use client";
import { Card, CardContent, CopyButton } from "@unkey/ui";
import React from "react";

export const LogSection = ({
  details,
  title,
}: {
  details: Record<string, React.ReactNode> | string;
  title: string;
}) => {
  // Helper function to get the text to copy
  const getTextToCopy = () => {
    let textToCopy: string;

    if (typeof details === "string") {
      textToCopy = details;
    } else {
      // Extract text content from React components
      textToCopy = Object.entries(details)
        .map(([key, value]) => {
          if (value === null || value === undefined) {
            return key;
          }
          // If it's a React element, try to extract text content
          if (typeof value === "object" && value !== null && "props" in value) {
            return `${key}: ${extractTextFromReactElement(value)}`;
          }
          return `${key}: ${value}`;
        })
        .join("\n");
    }
    return textToCopy;
  };

  // Helper function to extract text from React elements
  // This is used to extract text from React elements like TimestampInfo and Link components
  const extractTextFromReactElement = (element: React.ReactNode): string => {
    if (typeof element === "string" || typeof element === "number") {
      return String(element);
    }

    if (element === null || element === undefined) {
      return "";
    }

    // Handle React elements
    if (React.isValidElement(element)) {
      const reactElement = element as React.ReactElement<{
        value?: string | Date | number;
        children?: React.ReactNode;
        href?: string;
        title?: string;
      }>;

      // For TimestampInfo and similar components, check for a 'value' prop first
      if (reactElement.props.value) {
        // If value is a date/timestamp, format it appropriately
        if (reactElement.props.value instanceof Date) {
          return reactElement.props.value.toISOString();
        }
        return String(reactElement.props.value);
      }

      // Then check for children
      if (reactElement.props.children) {
        if (typeof reactElement.props.children === "string") {
          return reactElement.props.children;
        }
        if (Array.isArray(reactElement.props.children)) {
          return reactElement.props.children
            .map((child: React.ReactNode) => extractTextFromReactElement(child))
            .join("");
        }
        return extractTextFromReactElement(reactElement.props.children);
      }

      // For Link components, check for href or title
      if (reactElement.props.href || reactElement.props.title) {
        return reactElement.props.title || reactElement.props.href || "";
      }
    }

    return String(element);
  };

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">{title}</span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group">
          <pre className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {typeof details === "object"
              ? Object.entries(details).map((detail) => {
                  const [key, value] = detail;
                  return (
                    <div className="group flex items-center w-full p-[3px]" key={key}>
                      <span className="text-left text-accent-9 whitespace-nowrap">
                        {key}
                        {value ? ":" : ""}
                      </span>
                      <span className="ml-2 text-xs text-accent-12 truncate">{value}</span>
                    </div>
                  );
                })
              : details}
          </pre>

          <CopyButton
            value={getTextToCopy()}
            shape="square"
            variant="primary"
            size="2xlg"
            className="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4"
            aria-label="Copy content"
          />
        </CardContent>
      </Card>
    </div>
  );
};
