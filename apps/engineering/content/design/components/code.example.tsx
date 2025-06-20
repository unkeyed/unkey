"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Code, CopyButton, VisibleButton } from "@unkey/ui";
import { useState } from "react";
const EXAMPLE_SNIPPET = `curl -XPOST 'https://api.unkey.dev/v1/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
    -d '{
      "namespace": "demo_namespace",
    "identifier": "user_123",
    "limit": 10,
    "duration": 10000
}'`;
export function CodeExample() {
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4">
        <div>
          <div className=" items-center gap-4 w-full p-3">
            <Code className="relative w-full">
              {showKeyInSnippet
                ? EXAMPLE_SNIPPET
                : EXAMPLE_SNIPPET.replace("<UNKEY_ROOT_KEY>", "********")}
              <div className="absolute top-0 right-0 p-2 flex items-center gap-2">
                <VisibleButton
                  isVisible={showKeyInSnippet}
                  setIsVisible={(visible) => setShowKeyInSnippet(visible)}
                  title="Key Snippet"
                />

                <CopyButton
                  value={EXAMPLE_SNIPPET}
                  className="bg-transparent border border-grayA-6 rounded-md hover:bg-grayA-2"
                />
              </div>
            </Code>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CodeVariants() {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-4">
            <Code variant="default">Default Variant</Code>
            <Code variant="outline">Outline Variant</Code>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
