import { RenderComponentWithSnippet } from "@/app/components/render";
import { Code } from "@unkey/ui";

export function CodeExample() {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-4">
            <Code>const greeting = "Hello, World!";</Code>
            <Code>
              function add(a, b) {"{"} return a + b; {"}"}
            </Code>
            <Code>npm install @unkey/ui</Code>
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
