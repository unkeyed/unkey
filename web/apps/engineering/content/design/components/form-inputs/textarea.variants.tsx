"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { InputSearch } from "@unkey/icons";
import { Textarea } from "@unkey/ui";
import { useState } from "react";

export const TextareaDefaultVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea placeholder="All we have to decide is what to do with the time that is given us" />
    </RenderComponentWithSnippet>
  );
};

export const TextareaSuccessVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea variant="success" placeholder="Not all those who wander are lost" />
    </RenderComponentWithSnippet>
  );
};

export const TextareaWarningVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea variant="warning" placeholder="It's a dangerous business, going out your door" />
    </RenderComponentWithSnippet>
  );
};

export const TextareaErrorVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea variant="error" placeholder="One Ring to rule them all, One Ring to find them" />
    </RenderComponentWithSnippet>
  );
};

export const TextareaDisabledVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea
        disabled
        placeholder="Even the smallest person can change the course of the future"
      />
    </RenderComponentWithSnippet>
  );
};

export const TextareaWithDefaultValue = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea defaultValue="Speak friend and enter" placeholder="The password is mellon" />
    </RenderComponentWithSnippet>
  );
};

export const TextareaWithCharacterCount = () => {
  const [value, setValue] = useState(
    "There is some good in this world, and it's worth fighting for.",
  );
  const maxLength = 100;

  return (
    <RenderComponentWithSnippet>
      <div className="relative">
        <Textarea value={value} onChange={(e) => setValue(e.target.value)} maxLength={maxLength} />
        <div className="absolute bottom-2 right-3 text-xs text-gray-9">
          {value.length}/{maxLength}
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};

export const TextareaWithBothIcons = () => {
  return (
    <RenderComponentWithSnippet>
      <Textarea
        placeholder="Write your thoughts here"
        leftIcon={<InputSearch className="h-4 w-4" />}
        rightIcon={<InputSearch className="h-4 w-4" />}
      />
    </RenderComponentWithSnippet>
  );
};
