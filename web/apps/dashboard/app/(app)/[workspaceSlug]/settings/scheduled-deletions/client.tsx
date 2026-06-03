"use client";

import { WorkspaceNavbar } from "../workspace-navbar";
import { ScheduledDeletionsCardList } from "./components/card-list";
import { ScheduledDeletionsHeader } from "./components/header";

export function ScheduledDeletionsClient() {
  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "scheduled-deletions", text: "Scheduled Deletions" }} />
      <div className="w-full flex justify-center pb-20 px-8">
        <div className="flex flex-col w-full mt-6 gap-5" style={{ maxWidth: "960px" }}>
          <ScheduledDeletionsHeader />
          <ScheduledDeletionsCardList />
        </div>
      </div>
    </div>
  );
}
