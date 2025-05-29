"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { KeyboardButton } from "@unkey/ui";

export const Default = () => {


  return (
    <RenderComponentWithSnippet>
      <div className="flex w-300[px] h-32 border border-gray-6 rounded-md justify-end items-start p-3 gap-2">
        <span className="text-gray-9 text-[13px]">Default</span>
        <KeyboardButton shortcut="D" />
      </div>
    </RenderComponentWithSnippet>
  );
};

export const WithModifierKey = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex w-300[px] h-32 border border-gray-6 rounded-md justify-end items-start p-3 gap-2">
        <span className="text-gray-9 text-[13px]">Modifier Key</span>
        <KeyboardButton modifierKey="⌘" shortcut="K"  className="w-full m-0 p-0 gap-2"/>
      </div>
    </RenderComponentWithSnippet>
  );
};


// Example showing multiple shortcuts in a menu-like interface
export const ShortcutMenu = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="space-y-2 w-64 p-4 border rounded-lg">
        <div className="flex items-center justify-between">
          <span>Search</span>
          <KeyboardButton modifierKey="⌘" shortcut="K" />
        </div>
        <div className="flex items-center justify-between">
          <span>Save</span>
          <KeyboardButton modifierKey="⌘" shortcut="S" />
        </div>
        <div className="flex items-center justify-between">
          <span>Help</span>
          <KeyboardButton shortcut="?" />
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
