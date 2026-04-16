import { useState } from "react";
import { Button, Drover } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  return (
    <Preview>
      <Drover.Root>
        <Drover.Trigger asChild>
          <Button variant="outline">Open drover</Button>
        </Drover.Trigger>
        <Drover.Content>
          <div className="flex flex-col gap-2 p-3 min-w-[240px]">
            <div className="text-sm font-medium text-gray-12">
              Workspace actions
            </div>
            <div className="text-xs text-gray-11">
              Switches layout between popover and bottom drawer depending on
              viewport width.
            </div>
            <Drover.Close asChild>
              <Button variant="outline" size="sm" className="mt-2 self-start">
                Close
              </Button>
            </Drover.Close>
          </div>
        </Drover.Content>
      </Drover.Root>
    </Preview>
  );
}

export function ControlledExample() {
  const [open, setOpen] = useState(false);
  return (
    <Preview>
      <div className="flex items-center gap-3">
        <Button variant="outline" onClick={() => setOpen((v) => !v)}>
          {open ? "Close from parent" : "Open from parent"}
        </Button>
        <Drover.Root open={open} onOpenChange={setOpen}>
          <Drover.Trigger asChild>
            <Button variant="outline">Trigger</Button>
          </Drover.Trigger>
          <Drover.Content>
            <div className="flex flex-col gap-2 p-3 min-w-[240px]">
              <div className="text-sm font-medium text-gray-12">
                Controlled state
              </div>
              <div className="text-xs text-gray-11">
                Parent owns the open state; either button toggles it.
              </div>
            </div>
          </Drover.Content>
        </Drover.Root>
      </div>
    </Preview>
  );
}

export function NestedExample() {
  return (
    <Preview>
      <Drover.Root>
        <Drover.Trigger asChild>
          <Button variant="outline">Open parent</Button>
        </Drover.Trigger>
        <Drover.Content>
          <div className="flex flex-col gap-2 p-3 min-w-[260px]">
            <div className="text-sm font-medium text-gray-12">
              Parent drover
            </div>
            <div className="text-xs text-gray-11">
              On mobile, opening the nested drover slides the parent back so
              the child owns the full sheet.
            </div>
            <Drover.Nested>
              <Drover.Trigger asChild>
                <Button variant="outline" size="sm" className="self-start">
                  Open nested
                </Button>
              </Drover.Trigger>
              <Drover.Content>
                <div className="flex flex-col gap-2 p-3 min-w-[240px]">
                  <div className="text-sm font-medium text-gray-12">
                    Nested drover
                  </div>
                  <div className="text-xs text-gray-11">
                    Closing this closes the parent too.
                  </div>
                </div>
              </Drover.Content>
            </Drover.Nested>
          </div>
        </Drover.Content>
      </Drover.Root>
    </Preview>
  );
}
