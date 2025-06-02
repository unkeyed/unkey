"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { KeyboardButton } from "@unkey/ui";

export const Default = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex w-[300px] h-32 border border-gray-6 rounded-md justify-center items-center p-3 gap-2">
        <span className="text-gray-9 text-[13px]">Default</span>
        <KeyboardButton shortcut="D" />
        <span className="text-gray-9 text-[13px]">Modifier Key</span>
        <KeyboardButton modifierKey="⌘" shortcut="K" className="w-full m-0 p-0 gap-2" />
      </div>
    </RenderComponentWithSnippet>
  );
};
