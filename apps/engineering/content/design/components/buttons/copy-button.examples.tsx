import { RenderComponentWithSnippet } from "@/app/components/render";
import { CopyButton } from "@unkey/ui";
export const Default = () => {
  const apiKey = "uk_1234567890abcdef";
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-row justify-center gap-8">
        <div className="flex gap-2">
          <span>Basic usage:</span>
          <CopyButton value="Hello World!" />
        </div>
        <div className="flex gap-2 border px-2 py-1 rounded-md w-fit">
          <span className="font-mono text-sm">{apiKey}</span>
          <CopyButton value={apiKey} src="api-key-display" className="hover:bg-gray-100 rounded" />
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
