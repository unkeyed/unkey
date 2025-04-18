"use client";

import { CalendarClock, ChartPie, Code, Gauge, Key2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";

import { type NavItem, NavigableDialog } from "@/components/dialog-container/navigable-dialog";

export const CreateKeyDialog = () => {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  const settingsNavItems: NavItem[] = [
    {
      id: "general",
      label: "General Setup",
      icon: Key2,
      content: <div>Heheheh</div>,
    },
    {
      id: "ratelimit",
      label: "Ratelimit",
      icon: Gauge,
      content: <div>Asdsad</div>,
    },
    {
      id: "usage-limit",
      label: "Usage limit",
      icon: ChartPie,
      content: <div>Asdsad</div>,
    },
    {
      id: "expiration",
      label: "Expiration",
      icon: CalendarClock,
      content: <div>Asdsad</div>,
    },
    {
      id: "metadata",
      label: "Metadata",
      icon: Code,
      content: <div>Asdsad</div>,
    },
  ];

  return (
    <>
      <Button className="rounded-lg" onClick={() => setIsSettingsOpen(true)}>
        New Key
      </Button>
      <NavigableDialog
        isOpen={isSettingsOpen}
        onOpenChange={setIsSettingsOpen}
        title="New Key"
        subTitle="Create a custom API key with your own settings"
        items={settingsNavItems}
        footer={
          <div className="flex justify-center items-center w-full">
            <div className="flex flex-col items-center justify-center w-2/3 gap-2">
              <Button
                type="submit"
                form="identifier-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
              >
                Create new key
              </Button>
              <div className="text-xs text-gray-9">
                This key will be created immediately and ready-to-use right away
              </div>
            </div>
          </div>
        }
        dialogClassName="!min-w-[720px]"
      />
    </>
  );
};
