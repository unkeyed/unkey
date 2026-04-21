import { useRef, useState } from "react";
import {
  Button,
  ConfirmPopover,
  Dialog,
  DialogClose,
  DialogContainer,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
} from "@unkey/ui";
import { Gear, Key, User } from "@unkey/icons";
import { Preview } from "../../../components/Preview";

export function DialogExample() {
  const [open, setOpen] = useState(false);
  return (
    <Preview>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="outline">Open dialog</Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rotate API key</DialogTitle>
            <DialogDescription>
              A new secret will be issued and the current one will stop working
              within 60 seconds.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button color="danger" onClick={() => setOpen(false)}>
              Rotate key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Preview>
  );
}

export function DialogContainerExample() {
  const [open, setOpen] = useState(false);
  return (
    <Preview>
      <Button variant="outline" onClick={() => setOpen(true)}>
        Open container
      </Button>
      <DialogContainer
        isOpen={open}
        onOpenChange={setOpen}
        title="Invite teammate"
        subTitle="They will receive an email from ACME with a link to join."
        footer={
          <div className="flex w-full justify-end gap-2">
            <Button variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => setOpen(false)}>Send invite</Button>
          </div>
        }
      >
        <p className="text-sm text-gray-11">
          The container renders a pre-styled header, scrollable body, and
          bordered footer so you only supply the content and actions. Use it
          for most workspace dialogs.
        </p>
      </DialogContainer>
    </Preview>
  );
}

export function ConfirmationPopoverExample() {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const [deleted, setDeleted] = useState(false);

  return (
    <Preview>
      <div className="flex flex-col items-center gap-3">
        <Button
          ref={triggerRef}
          variant="outline"
          color="danger"
          onClick={() => setOpen(true)}
          disabled={deleted}
        >
          {deleted ? "Deleted" : "Delete key"}
        </Button>
        <ConfirmPopover
          isOpen={open}
          onOpenChange={setOpen}
          onConfirm={() => setDeleted(true)}
          triggerRef={triggerRef}
          title="Delete this key?"
          description="This action cannot be undone. Any service using this key will start failing."
          confirmButtonText="Delete"
          variant="danger"
        />
        {deleted && (
          <button
            type="button"
            onClick={() => setDeleted(false)}
            className="text-[11px] uppercase tracking-[0.2em] text-gray-9 hover:text-gray-12"
          >
            Reset
          </button>
        )}
      </div>
    </Preview>
  );
}

const NAV_ITEMS = [
  { id: "profile" as const, label: "Profile", icon: User },
  { id: "keys" as const, label: "API keys", icon: Key },
  { id: "settings" as const, label: "Settings", icon: Gear },
];

const CONTENT_ITEMS = [
  {
    id: "profile" as const,
    content: (
      <div className="flex flex-col gap-2">
        <h3 className="text-base font-medium text-gray-12">Profile</h3>
        <p className="text-sm text-gray-11">
          Your display name and avatar shown across the ACME workspace.
        </p>
      </div>
    ),
  },
  {
    id: "keys" as const,
    content: (
      <div className="flex flex-col gap-2">
        <h3 className="text-base font-medium text-gray-12">API keys</h3>
        <p className="text-sm text-gray-11">
          Manage personal access tokens and rotation schedules. Switching
          panels here preserves the parent dialog's open state.
        </p>
      </div>
    ),
  },
  {
    id: "settings" as const,
    content: (
      <div className="flex flex-col gap-2">
        <h3 className="text-base font-medium text-gray-12">Settings</h3>
        <p className="text-sm text-gray-11">
          Notifications, defaults, and everything else that does not fit the
          other two panes.
        </p>
      </div>
    ),
  },
];

export function NavigableDialogExample() {
  const [open, setOpen] = useState(false);
  return (
    <Preview>
      <Button variant="outline" onClick={() => setOpen(true)}>
        Open stepped dialog
      </Button>
      <NavigableDialogRoot
        isOpen={open}
        onOpenChange={setOpen}
        dialogClassName="w-[90%] md:w-[70%] xl:w-[60%] max-w-[720px]"
      >
        <NavigableDialogHeader
          title="Account"
          subTitle="Switch between panes on the left"
        />
        <NavigableDialogBody>
          <NavigableDialogNav items={NAV_ITEMS} />
          <NavigableDialogContent items={CONTENT_ITEMS} />
        </NavigableDialogBody>
        <NavigableDialogFooter>
          <div className="flex w-full justify-end gap-2">
            <Button variant="outline" onClick={() => setOpen(false)}>
              Close
            </Button>
            <Button onClick={() => setOpen(false)}>Save changes</Button>
          </div>
        </NavigableDialogFooter>
      </NavigableDialogRoot>
    </Preview>
  );
}
