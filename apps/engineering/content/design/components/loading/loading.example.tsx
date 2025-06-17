"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Loading } from "@unkey/ui";

export function LoadingExample() {
  return (
    <RenderComponentWithSnippet>
      <div className="flex items-center gap-4 justify-center">
        <Loading />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CustomSizeAndDuration() {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4 justify-center items-center">
        <div className="flex items-center gap-4 text-[#00FFFF]">
          <Loading size={48} duration={"250ms"} />
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function DotsLineExample() {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4 justify-center items-center">
        <div className="flex items-center gap-4 text-[#00FFFF]">
          <Loading type="dots" />
        </div>
        <div className="flex items-center gap-4 text-[#00FFFF]">
          <Loading type="dots" duration={"400ms"} />
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
