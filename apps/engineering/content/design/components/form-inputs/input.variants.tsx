"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { InputSearch } from "@unkey/icons";
import { Input } from "@unkey/ui";
import { EyeIcon, EyeOff } from "lucide-react";
import { useState } from "react";

export const InputDefaultVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Input placeholder="All we have to decide is what to do with the time that is given us" />
    </RenderComponentWithSnippet>
  );
};

export const InputSuccessVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Input variant="success" placeholder="Not all those who wander are lost" />
    </RenderComponentWithSnippet>
  );
};

export const InputWarningVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Input variant="warning" placeholder="It's a dangerous business, going out your door" />
    </RenderComponentWithSnippet>
  );
};

export const InputErrorVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Input variant="error" placeholder="One Ring to rule them all, One Ring to find them" />
    </RenderComponentWithSnippet>
  );
};

export const InputDisabledVariant = () => {
  return (
    <RenderComponentWithSnippet>
      <Input disabled placeholder="Even the smallest person can change the course of the future" />
    </RenderComponentWithSnippet>
  );
};

export const InputWithDefaultValue = () => {
  return (
    <RenderComponentWithSnippet>
      <Input defaultValue="Speak friend and enter" placeholder="The password is mellon" />
    </RenderComponentWithSnippet>
  );
};

export const InputWithPasswordToggle = () => {
  const [showPassword, setShowPassword] = useState(false);

  return (
    <RenderComponentWithSnippet>
      <Input
        type={showPassword ? "text" : "password"}
        placeholder="Speak friend and enter"
        rightIcon={
          <button
            type="button"
            onClick={() => setShowPassword(!showPassword)}
            className="focus:outline-none"
          >
            {showPassword ? <EyeIcon className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
          </button>
        }
      />
    </RenderComponentWithSnippet>
  );
};

export const InputWithBothIcons = () => {
  return (
    <RenderComponentWithSnippet>
      <Input
        placeholder="Search in emails"
        leftIcon={<InputSearch className="h-4 w-4" />}
        rightIcon={<InputSearch className="h-4 w-4" />}
      />
    </RenderComponentWithSnippet>
  );
};

export const InputWithPrefix = () => {
  return (
    <RenderComponentWithSnippet>
      <Input prefix="https://" placeholder="example.com" />
    </RenderComponentWithSnippet>
  );
};

export const InputWithPrefixAndIcon = () => {
  return (
    <RenderComponentWithSnippet>
      <Input
        prefix="$"
        placeholder="0.00"
        rightIcon={<span className="text-xs opacity-60">USD</span>}
      />
    </RenderComponentWithSnippet>
  );
};

export const InputWithPrefixVariants = () => {
  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-4">
        <Input prefix="@" placeholder="username" />
        <Input prefix="+" placeholder="1 (555) 123-4567" variant="success" />
        <Input prefix="ref_" placeholder="transaction_id" variant="warning" />
        <Input prefix="key_" placeholder="api_key_here" variant="error" />
      </div>
    </RenderComponentWithSnippet>
  );
};
