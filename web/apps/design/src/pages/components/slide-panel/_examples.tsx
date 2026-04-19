import { Button, SlidePanel } from "@unkey/ui";
import { useState } from "react";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Preview>
      <Button variant="outline" onClick={() => setIsOpen(true)}>
        Open panel
      </Button>
      <SlidePanel.Root isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <SlidePanel.Header>
          <div className="flex flex-col">
            <span className="text-gray-12 font-medium text-base leading-8">
              Create API key
            </span>
            <span className="text-gray-11 text-[13px] leading-5">
              Configure a new key for the ACME workspace.
            </span>
          </div>
          <SlidePanel.Close
            aria-label="Close panel"
            className="inline-flex items-center justify-center h-8 px-3 rounded-md text-sm text-gray-11 hover:bg-grayA-3 transition-colors"
          >
            Close
          </SlidePanel.Close>
        </SlidePanel.Header>
        <SlidePanel.Content>
          <div className="px-8 py-6 text-sm text-gray-11">
            <p>
              Panel content goes here. The content area fills the remaining
              height and scrolls independently when it overflows.
            </p>
          </div>
        </SlidePanel.Content>
        <SlidePanel.Footer>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={() => setIsOpen(false)}>
              Create
            </Button>
          </div>
        </SlidePanel.Footer>
      </SlidePanel.Root>
    </Preview>
  );
}

export function SideExample() {
  const [openRight, setOpenRight] = useState(false);
  const [openLeft, setOpenLeft] = useState(false);

  return (
    <Preview>
      <Button variant="outline" onClick={() => setOpenRight(true)}>
        Slide from right
      </Button>
      <Button variant="outline" onClick={() => setOpenLeft(true)}>
        Slide from left
      </Button>
      <SlidePanel.Root
        isOpen={openRight}
        onClose={() => setOpenRight(false)}
        side="right"
      >
        <SlidePanel.Header>
          <span className="text-gray-12 font-medium text-base leading-8">
            Right-anchored
          </span>
          <SlidePanel.Close
            aria-label="Close panel"
            className="inline-flex items-center justify-center h-8 px-3 rounded-md text-sm text-gray-11 hover:bg-grayA-3 transition-colors"
          >
            Close
          </SlidePanel.Close>
        </SlidePanel.Header>
        <SlidePanel.Content>
          <div className="px-8 py-6 text-sm text-gray-11">
            The default. Slides in from the right edge and tucks back out when
            dismissed.
          </div>
        </SlidePanel.Content>
      </SlidePanel.Root>
      <SlidePanel.Root
        isOpen={openLeft}
        onClose={() => setOpenLeft(false)}
        side="left"
      >
        <SlidePanel.Header>
          <span className="text-gray-12 font-medium text-base leading-8">
            Left-anchored
          </span>
          <SlidePanel.Close
            aria-label="Close panel"
            className="inline-flex items-center justify-center h-8 px-3 rounded-md text-sm text-gray-11 hover:bg-grayA-3 transition-colors"
          >
            Close
          </SlidePanel.Close>
        </SlidePanel.Header>
        <SlidePanel.Content>
          <div className="px-8 py-6 text-sm text-gray-11">
            Mirrors the default behaviour against the left edge. Useful when
            the triggering content lives on the right side of the page.
          </div>
        </SlidePanel.Content>
      </SlidePanel.Root>
    </Preview>
  );
}

export function WidthExample() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Preview>
      <Button variant="outline" onClick={() => setIsOpen(true)}>
        Open wide panel
      </Button>
      <SlidePanel.Root
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        widthClassName="w-[min(100vw-1.5rem,64rem)]"
      >
        <SlidePanel.Header>
          <span className="text-gray-12 font-medium text-base leading-8">
            Request inspector
          </span>
          <SlidePanel.Close
            aria-label="Close panel"
            className="inline-flex items-center justify-center h-8 px-3 rounded-md text-sm text-gray-11 hover:bg-grayA-3 transition-colors"
          >
            Close
          </SlidePanel.Close>
        </SlidePanel.Header>
        <SlidePanel.Content>
          <div className="px-8 py-6 text-sm text-gray-11">
            Override <code>widthClassName</code> when the default 43.75rem
            (<code>w-175</code>) isn't enough. Wide panels work well for log
            inspectors, diff views, and side-by-side editors.
          </div>
        </SlidePanel.Content>
      </SlidePanel.Root>
    </Preview>
  );
}

export function TopOffsetExample() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Preview>
      <Button variant="outline" onClick={() => setIsOpen(true)}>
        Open under a header
      </Button>
      <SlidePanel.Root
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        topOffset={64}
      >
        <SlidePanel.Header>
          <span className="text-gray-12 font-medium text-base leading-8">
            Offset panel
          </span>
          <SlidePanel.Close
            aria-label="Close panel"
            className="inline-flex items-center justify-center h-8 px-3 rounded-md text-sm text-gray-11 hover:bg-grayA-3 transition-colors"
          >
            Close
          </SlidePanel.Close>
        </SlidePanel.Header>
        <SlidePanel.Content>
          <div className="px-8 py-6 text-sm text-gray-11">
            Pass <code>topOffset</code> (in pixels) to drop the panel below a
            fixed page header or tab bar. The panel measures its height from
            the offset, so the top of the panel aligns with the content
            underneath.
          </div>
        </SlidePanel.Content>
      </SlidePanel.Root>
    </Preview>
  );
}
