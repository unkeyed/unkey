import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  return (
    <Preview>
      <Tabs defaultValue="overview" className="w-[420px]">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="keys">Keys</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>
        <TabsContent value="overview">
          <p className="text-sm text-gray-11">
            2,431 keys issued against the ACME production API today.
          </p>
        </TabsContent>
        <TabsContent value="keys">
          <p className="text-sm text-gray-11">
            Browse, rotate, and revoke individual API keys.
          </p>
        </TabsContent>
        <TabsContent value="settings">
          <p className="text-sm text-gray-11">
            Workspace-level configuration for the ACME API.
          </p>
        </TabsContent>
      </Tabs>
    </Preview>
  );
}

export function ControlledExample() {
  const [value, setValue] = useState("overview");
  return (
    <Preview>
      <div className="flex w-[420px] flex-col gap-3">
        <div className="text-xs uppercase tracking-[0.2em] text-gray-9">
          Active: {value}
        </div>
        <Tabs value={value} onValueChange={setValue}>
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="keys">Keys</TabsTrigger>
            <TabsTrigger value="settings">Settings</TabsTrigger>
          </TabsList>
          <TabsContent value="overview">
            <p className="text-sm text-gray-11">
              The parent owns the active tab; changing state from outside also
              updates the tabs.
            </p>
          </TabsContent>
          <TabsContent value="keys">
            <p className="text-sm text-gray-11">
              Pair `value` with `onValueChange` to drive Tabs from a URL query
              param, a store, or a form.
            </p>
          </TabsContent>
          <TabsContent value="settings">
            <p className="text-sm text-gray-11">
              Use controlled mode when another part of the UI needs to read or
              set the current tab.
            </p>
          </TabsContent>
        </Tabs>
      </div>
    </Preview>
  );
}

export function OrientationExample() {
  return (
    <Preview>
      <Tabs
        defaultValue="overview"
        orientation="vertical"
        className="flex w-[520px] gap-4"
      >
        <TabsList className="flex h-auto flex-col items-stretch">
          <TabsTrigger value="overview" className="justify-start">
            Overview
          </TabsTrigger>
          <TabsTrigger value="keys" className="justify-start">
            Keys
          </TabsTrigger>
          <TabsTrigger value="settings" className="justify-start">
            Settings
          </TabsTrigger>
        </TabsList>
        <div className="flex-1">
          <TabsContent value="overview" className="mt-0">
            <p className="text-sm text-gray-11">
              Vertical tabs suit settings panes and dense navigation where a
              horizontal row would wrap.
            </p>
          </TabsContent>
          <TabsContent value="keys" className="mt-0">
            <p className="text-sm text-gray-11">
              Arrow-key navigation follows the orientation: up/down for
              vertical, left/right for horizontal.
            </p>
          </TabsContent>
          <TabsContent value="settings" className="mt-0">
            <p className="text-sm text-gray-11">
              Override `TabsList` layout classes to render the list as a
              column.
            </p>
          </TabsContent>
        </div>
      </Tabs>
    </Preview>
  );
}
