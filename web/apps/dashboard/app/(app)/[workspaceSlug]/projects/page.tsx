"use client";

import { ProjectsList } from "./_components/list";

// TODO(deploy): REMOVE BEFORE MERGE — temporarily renders only the empty state
// (no navbar/controls) for design review of the "Meet Unkey Deploy" empty page.
export default function ProjectsPage() {
  return (
    <div className="flex flex-1 w-full">
      <ProjectsList />
    </div>
  );
}
