import { RenderComponentWithSnippet } from "@/app/components/render";
import { Separator } from "@unkey/ui";

export function HorizontalSeparator() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-6">
  {/* Horizontal separator */}
  <div>
    <h4 className="text-sm font-medium leading-none text-center">Horizontal</h4>
    <div className="my-4">Some content above</div>
    <Separator />
    <div className="my-4">Some content below</div>
  </div>
</div>`}
    >
      <div className="space-y-6">
        {/* Horizontal separator */}
        <div>
          <h4 className="text-sm font-medium leading-none text-center">Horizontal</h4>
          <div className="my-4">Some content above</div>
          <Separator />
          <div className="my-4">Some content below</div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function VerticalSeparator() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div>
  <div className="flex flex-col gap-4 justify-center items-center">
    <h4 className="text-sm font-medium leading-none">Vertical</h4>
  </div>
  <div className="space-y-6 flex flex-row gap-4 w-full">
    {/* Vertical separator */}

    <div className="flex flex-col gap-4">
      <div className="flex  gap-4">Left</div>
      <div className="flex  gap-4">Left Content</div>
      <div className="flex  gap-4">Left Content</div>
    </div>

    <Separator orientation="vertical" />
    <div className="flex flex-col gap-4">
      <div className="flex  gap-4">Right</div>
      <div className="flex  gap-4">Right Content</div>
      <div className="flex  gap-4">Right Content</div>
    </div>
  </div>
</div>`}
    >
      <div>
        <div className="flex flex-col gap-4 justify-center items-center">
          <h4 className="text-sm font-medium leading-none">Vertical</h4>
        </div>
        <div className="space-y-6 flex flex-row gap-4 w-full">
          {/* Vertical separator */}

          <div className="flex flex-col gap-4">
            <div className="flex  gap-4">Left</div>
            <div className="flex  gap-4">Left Content</div>
            <div className="flex  gap-4">Left Content</div>
          </div>

          <Separator orientation="vertical" />
          <div className="flex flex-col gap-4">
            <div className="flex  gap-4">Right</div>
            <div className="flex  gap-4">Right Content</div>
            <div className="flex  gap-4">Right Content</div>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function DecorativeSeparator() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-6">
  {/* Horizontal separator */}
  <div>
    <h4 className="text-sm font-medium leading-none text-center">Decorative</h4>
    <div className="my-4">Some content above</div>
    <Separator decorative />
    <div className="my-4">Some content below</div>
  </div>
</div>`}
    >
      <div className="space-y-6">
        {/* Horizontal separator */}
        <div>
          <h4 className="text-sm font-medium leading-none text-center">Decorative</h4>
          <div className="my-4">Some content above</div>
          <Separator decorative />
          <div className="my-4">Some content below</div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
