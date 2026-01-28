"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Book2, Key } from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import {
  Button,
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
} from "@unkey/ui";
import { memo, useState } from "react";
import type { FC } from "react";
import type React from "react";

// Memoize static content to prevent unnecessary re-renders
const TabContent = memo(({ title, description }: { title: string; description: string }) => (
  <div className="p-4">
    <h3 className="text-lg font-medium mb-2">{title}</h3>
    <p className="text-gray-11">{description}</p>
  </div>
));
TabContent.displayName = "TabContent";

type TabId = "docs" | "security";

// Pre-define navigation items to prevent recreation on each render
const NAV_ITEMS: Array<{
  id: TabId;
  label: string;
  icon: FC<IconProps>;
}> = [
  {
    id: "docs",
    label: "Documentation",
    icon: Book2,
  },
  {
    id: "security",
    label: "Security",
    icon: Key,
  },
];

// Pre-define content items using the memoized TabContent
const CONTENT_ITEMS: Array<{
  id: TabId;
  content: React.ReactElement;
}> = [
  {
    id: "docs",
    content: (
      <TabContent
        title="Documentation"
        description="Access comprehensive guides and documentation for the NavigableDialog component."
      />
    ),
  },
  {
    id: "security",
    content: (
      <TabContent
        title="Security"
        description="Review security settings and configurations for your application."
      />
    ),
  },
];

// Main example component
export const NavigableDialogExample = memo(() => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex justify-center">
  <Button onClick={() => setIsOpen(true)}>Open Example Dialog</Button>

  <NavigableDialogRoot isOpen={isOpen} onOpenChange={setIsOpen} preventAutoFocus>
    <NavigableDialogHeader
      title="NavigableDialog Example"
      subTitle="A simple demonstration"
    />

    <NavigableDialogBody className="flex text-warning-5">
      <NavigableDialogNav items={NAV_ITEMS} />
      <NavigableDialogContent items={CONTENT_ITEMS} />
    </NavigableDialogBody>

    <NavigableDialogFooter>
      <Button onClick={() => setIsOpen(false)}>Close</Button>
    </NavigableDialogFooter>
  </NavigableDialogRoot>
</div>`}
    >
      <div className="flex justify-center">
        <Button onClick={() => setIsOpen(true)}>Open Example Dialog</Button>

        <NavigableDialogRoot isOpen={isOpen} onOpenChange={setIsOpen} preventAutoFocus>
          <NavigableDialogHeader
            title="NavigableDialog Example"
            subTitle="A simple demonstration"
          />

          <NavigableDialogBody className="flex text-warning-5">
            <NavigableDialogNav items={NAV_ITEMS} />
            <NavigableDialogContent items={CONTENT_ITEMS} />
          </NavigableDialogBody>

          <NavigableDialogFooter>
            <Button onClick={() => setIsOpen(false)}>Close</Button>
          </NavigableDialogFooter>
        </NavigableDialogRoot>
      </div>
    </RenderComponentWithSnippet>
  );
});

NavigableDialogExample.displayName = "NavigableDialogExample";
