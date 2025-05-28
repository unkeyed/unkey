"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button, KeyboardButton } from "@unkey/ui";
import { useState } from "react";

export const Default = () => {
  const [activated, setActivated] = useState(false);
  const message = activated ? "Key has been activated" : "Press k";

  const handleClick = () => {
    setActivated(!activated);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="flex items-center gap-2 bg-white">
        <Button variant="ghost" size="md" title="Press k" onClick={handleClick}>
          <KeyboardButton shortcut="f" />
        </Button>
        <span className="text-sm">{message}</span>
      </div>
    </RenderComponentWithSnippet>
  );
};

export const WithModifierKey = () => {
  return (
    <RenderComponentWithSnippet>
      <KeyboardButton modifierKey="⌘" shortcut="K" />
    </RenderComponentWithSnippet>
  );
};

export const WithShiftModifier = () => {
  return (
    <RenderComponentWithSnippet>
      <KeyboardButton modifierKey="⇧" shortcut="/" />
    </RenderComponentWithSnippet>
  );
};

export const WithControlModifier = () => {
  return (
    <RenderComponentWithSnippet>
      <KeyboardButton modifierKey="CTRL" shortcut="S" />
    </RenderComponentWithSnippet>
  );
};

export const WithOptionModifier = () => {
  return (
    <RenderComponentWithSnippet>
      <KeyboardButton modifierKey="⌥" shortcut="B" />
    </RenderComponentWithSnippet>
  );
};

export const CustomStyle = () => {
  return (
    <RenderComponentWithSnippet>
      <KeyboardButton
        shortcut="P"
        className="bg-blue-100 dark:bg-blue-900 text-blue-900 dark:text-blue-100"
      />
    </RenderComponentWithSnippet>
  );
};

export const KeyboardShortcutGroup = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex items-center gap-2">
        <KeyboardButton modifierKey="⌘" shortcut="K" />
        <span className="text-sm">or</span>
        <KeyboardButton modifierKey="CTRL" shortcut="K" />
      </div>
    </RenderComponentWithSnippet>
  );
};

// Example showing how to use the component in a search interface
export const SearchInterface = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex items-center justify-between w-full max-w-md px-4 py-2 border rounded-lg">
        <span className="text-sm text-gray-600">Quick search...</span>
        <KeyboardButton modifierKey="⌘" shortcut="K" />
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
