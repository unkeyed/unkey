"use client";

import { NamespaceListControlCloud } from "./control-cloud";
import { NamespaceListControls } from "./controls";
import { NamespaceList } from "./list";

export const NamespaceListClient = () => {
  return (
    <div className="flex flex-col relative">
      <NamespaceListControls />
      <NamespaceListControlCloud />
      <NamespaceList />
    </div>
  );
};
