import { RenderComponentWithSnippet } from "@/app/components/render";
import { CopyButton } from "@unkey/ui";

export const Default = () => {
  const apiKey = "uk_1234567890abcdef";

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div className="flex flex-row justify-center gap-8">
    <div className="flex gap-2 items-center">
      <span>Basic usage:</span>
      <CopyButton value="Hello World!" />
    </div>
    <div className="flex gap-2 border px-2 py-1 rounded-md w-fit items-center">
      <span className="font-mono text-sm">{apiKey}</span>
      <CopyButton variant="ghost" value={apiKey} src="api-key-display" />
    </div>
  </div>

  <div className="flex flex-col gap-4">
    <div className="flex gap-2 items-center">
      <span>Different variants:</span>
      <CopyButton value="Outline variant (default)" variant="outline" />
      <CopyButton value="Ghost variant" variant="ghost" />
      <CopyButton value="Primary variant" variant="primary" />
    </div>

    <div className="flex gap-2 items-center">
      <span>With custom styling:</span>
      <CopyButton
        value="Custom styled button"
        className="bg-blue-100 hover:bg-blue-200 border-blue-300"
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div className="flex flex-row justify-center gap-8">
          <div className="flex gap-2 items-center">
            <span>Basic usage:</span>
            <CopyButton value="Hello World!" />
          </div>
          <div className="flex gap-2 border px-2 py-1 rounded-md w-fit items-center">
            <span className="font-mono text-sm">{apiKey}</span>
            <CopyButton variant="ghost" value={apiKey} src="api-key-display" />
          </div>
        </div>

        <div className="flex flex-col gap-4">
          <div className="flex gap-2 items-center">
            <span>Different variants:</span>
            <CopyButton value="Outline variant (default)" variant="outline" />
            <CopyButton value="Ghost variant" variant="ghost" />
            <CopyButton value="Primary variant" variant="primary" />
          </div>

          <div className="flex gap-2 items-center">
            <span>With custom styling:</span>
            <CopyButton
              value="Custom styled button"
              className="bg-blue-100 hover:bg-blue-200 border-blue-300"
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
