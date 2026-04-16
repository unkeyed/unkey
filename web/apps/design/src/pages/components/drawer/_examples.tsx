import { Button, Drawer } from "@unkey/ui";
import { useState } from "react";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  return (
    <Preview>
      <Drawer.Root>
        <Drawer.Trigger asChild>
          <Button variant="outline">Open drawer</Button>
        </Drawer.Trigger>
        <Drawer.Content>
          <div className="p-6 flex flex-col gap-2">
            <Drawer.Title className="text-lg font-medium text-gray-12">
              Rotate key
            </Drawer.Title>
            <Drawer.Description className="text-sm text-gray-11">
              Generate a new secret for this key. The old secret keeps working
              for another 24 hours before it is revoked.
            </Drawer.Description>
            <div className="flex justify-end gap-2 pt-4">
              <Drawer.Trigger asChild>
                <Button variant="ghost">Cancel</Button>
              </Drawer.Trigger>
              <Button variant="primary">Rotate</Button>
            </div>
          </div>
        </Drawer.Content>
      </Drawer.Root>
    </Preview>
  );
}

export function ControlledExample() {
  const [open, setOpen] = useState(false);

  return (
    <Preview>
      <div className="flex flex-col items-center gap-2">
        <Button variant="outline" onClick={() => setOpen(true)}>
          Open from parent state
        </Button>
        <span className="text-[11px] uppercase tracking-[0.18em] text-gray-9">
          Drawer is {open ? "open" : "closed"}
        </span>
      </div>
      <Drawer.Root open={open} onOpenChange={setOpen}>
        <Drawer.Content>
          <div className="p-6 flex flex-col gap-3">
            <Drawer.Title className="text-lg font-medium text-gray-12">
              Controlled drawer
            </Drawer.Title>
            <Drawer.Description className="text-sm text-gray-11">
              The parent component owns the open state. Drag down, hit escape,
              or press the button to close.
            </Drawer.Description>
            <div className="flex justify-end pt-4">
              <Button variant="outline" onClick={() => setOpen(false)}>
                Close
              </Button>
            </div>
          </div>
        </Drawer.Content>
      </Drawer.Root>
    </Preview>
  );
}

export function NestedExample() {
  return (
    <Preview>
      <Drawer.Root>
        <Drawer.Trigger asChild>
          <Button variant="outline">Open parent</Button>
        </Drawer.Trigger>
        <Drawer.Content>
          <div className="p-6 flex flex-col gap-3">
            <Drawer.Title className="text-lg font-medium text-gray-12">
              Workspace settings
            </Drawer.Title>
            <Drawer.Description className="text-sm text-gray-11">
              Opening a nested drawer pushes this one back and stacks the new
              one on top, matching mobile platform conventions.
            </Drawer.Description>
            <Drawer.Nested>
              <Drawer.Trigger asChild>
                <Button variant="outline" className="self-start">
                  Open nested
                </Button>
              </Drawer.Trigger>
              <Drawer.Content>
                <div className="p-6 flex flex-col gap-2">
                  <Drawer.Title className="text-lg font-medium text-gray-12">
                    Danger zone
                  </Drawer.Title>
                  <Drawer.Description className="text-sm text-gray-11">
                    Nested drawers share the same overlay and dismissal
                    gestures as the parent.
                  </Drawer.Description>
                </div>
              </Drawer.Content>
            </Drawer.Nested>
          </div>
        </Drawer.Content>
      </Drawer.Root>
    </Preview>
  );
}
