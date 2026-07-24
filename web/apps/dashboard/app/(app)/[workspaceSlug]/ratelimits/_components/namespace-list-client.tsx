"use client";

import { NamespaceListControls } from "./controls";
import { NamespaceList } from "./list";

export const NamespaceListClient = () => {
  return (
    <div className="flex flex-col gap-4">
      <NamespaceListControls />
      <NamespaceList />
    </div>
  );
};
