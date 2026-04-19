import { useState } from "react";
import {
  Button,
  Input,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  return (
    <Preview>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline">Rename workspace</Button>
        </PopoverTrigger>
        <PopoverContent>
          <div className="flex flex-col gap-3">
            <label className="text-xs font-medium text-gray-11" htmlFor="ws-name">
              New name
            </label>
            <Input id="ws-name" defaultValue="acme-prod" />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm">
                Cancel
              </Button>
              <Button variant="primary" size="sm">
                Save
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </Preview>
  );
}

export function PositioningExample() {
  return (
    <Preview>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline">Top</Button>
        </PopoverTrigger>
        <PopoverContent side="top">
          <p className="text-sm text-gray-12">
            Anchored to the top edge of the trigger.
          </p>
        </PopoverContent>
      </Popover>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline">Right</Button>
        </PopoverTrigger>
        <PopoverContent side="right">
          <p className="text-sm text-gray-12">
            Anchored to the right edge of the trigger.
          </p>
        </PopoverContent>
      </Popover>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline">Bottom, end-aligned</Button>
        </PopoverTrigger>
        <PopoverContent side="bottom" align="end" sideOffset={8}>
          <p className="text-sm text-gray-12">
            Offset by 8px, aligned to the trigger's end.
          </p>
        </PopoverContent>
      </Popover>
    </Preview>
  );
}

export function ControlledExample() {
  const [open, setOpen] = useState(false);
  return (
    <Preview>
      <div className="flex items-center gap-3">
        <Button variant="outline" onClick={() => setOpen((v) => !v)}>
          {open ? "Close from outside" : "Open from outside"}
        </Button>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button variant="primary">Trigger</Button>
          </PopoverTrigger>
          <PopoverContent>
            <div className="flex flex-col gap-2">
              <p className="text-sm text-gray-12">
                Open state is held by the parent. Either button flips it.
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setOpen(false)}
              >
                Dismiss
              </Button>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </Preview>
  );
}
