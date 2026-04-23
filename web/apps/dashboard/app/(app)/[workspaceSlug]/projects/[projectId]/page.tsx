"use client";

import { useProjectItems } from "@/hooks/use-project-items";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type ProjectItem, type ProjectItemType, sortByTypeGroup } from "@/lib/project-items";
import { Cube, Database, Envelope, Lock, Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput, Loading } from "@unkey/ui";
import Link from "next/link";
import { useMemo, useState } from "react";
import { useProjectData } from "./(overview)/data-provider";

export default function ProjectOverviewPage() {
  const { projectId, project, isProjectLoading } = useProjectData();
  const workspace = useWorkspaceNavigation();
  const { items, addItem } = useProjectItems(projectId);
  const [addOpen, setAddOpen] = useState(false);

  const sorted = useMemo(() => sortByTypeGroup(items), [items]);

  return (
    <div className="flex flex-col gap-8 p-6 w-full max-w-[1200px] mx-auto">
      <header className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-accent-12">
            {isProjectLoading ? <Loading type="spinner" /> : (project?.name ?? "Project")}
          </h1>
          <p className="text-sm text-gray-11 mt-1">
            Apps and supplementary services inside this project.
          </p>
        </div>
        <Button variant="primary" size="md" onClick={() => setAddOpen(true)}>
          <Plus /> Add item
        </Button>
      </header>

      {sorted.length === 0 ? (
        <EmptyState onAdd={() => setAddOpen(true)} />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {sorted.map((item) => (
            <ItemCard key={item.id} item={item} href={itemHref(workspace.slug, projectId, item)} />
          ))}
        </div>
      )}

      <AddItemDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        onSubmit={(input) => {
          addItem(input);
          setAddOpen(false);
        }}
      />
    </div>
  );
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="border border-dashed border-grayA-4 rounded-2xl p-12 flex flex-col items-center gap-3">
      <p className="text-sm text-gray-11">This project has nothing in it yet.</p>
      <Button variant="primary" onClick={onAdd}>
        <Plus /> Add your first item
      </Button>
    </div>
  );
}

function ItemCard({ item, href }: { item: ProjectItem; href: string }) {
  const Icon = iconForType(item.type);
  return (
    <Link
      href={href}
      className="group relative flex flex-col gap-4 p-5 border border-grayA-4 hover:border-grayA-7 rounded-2xl transition-colors"
    >
      <div className="flex items-center gap-3">
        <div className="size-10 bg-gray-3 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20 dark:ring-1 dark:ring-gray-4 dark:shadow-none">
          <Icon iconSize="xl-medium" className="size-5" />
        </div>
        <div className="min-w-0 flex flex-col gap-1">
          <span className="font-medium text-sm text-accent-12 truncate group-hover:underline">
            {item.name}
          </span>
          <span className="text-xs text-gray-11 truncate">{item.slug}</span>
        </div>
      </div>
    </Link>
  );
}

type AddItemDialogProps = {
  open: boolean;
  onOpenChange: (value: boolean) => void;
  onSubmit: (input: { type: ProjectItemType; name: string }) => void;
};

function AddItemDialog({ open, onOpenChange, onSubmit }: AddItemDialogProps) {
  const [type, setType] = useState<ProjectItemType>("app");
  const [name, setName] = useState("");

  const reset = () => {
    setType("app");
    setName("");
  };

  const handleOpenChange = (value: boolean) => {
    if (!value) {
      reset();
    }
    onOpenChange(value);
  };

  const canSubmit = name.trim().length > 0;

  return (
    <DialogContainer
      isOpen={open}
      onOpenChange={handleOpenChange}
      title="Add item"
      subTitle="Pick a type and give it a name. This only affects your local view."
      footer={
        <Button
          variant="primary"
          size="xlg"
          disabled={!canSubmit}
          onClick={() => {
            if (!canSubmit) {
              return;
            }
            onSubmit({ type, name: name.trim() });
            reset();
          }}
        >
          Add item
        </Button>
      }
    >
      <div className="flex flex-col gap-4 px-6 py-4">
        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium text-accent-12">Type</span>
          <div className="grid grid-cols-2 gap-2">
            {(["app", "database", "queue", "vault"] as ProjectItemType[]).map((t) => {
              const Icon = iconForType(t);
              const selected = type === t;
              return (
                <button
                  key={t}
                  type="button"
                  onClick={() => setType(t)}
                  className={`flex items-center gap-2 p-3 rounded-lg border text-sm transition-colors ${
                    selected
                      ? "border-accent-12 bg-gray-3 text-accent-12"
                      : "border-grayA-4 text-gray-11 hover:border-grayA-6 hover:text-accent-12"
                  }`}
                >
                  <Icon iconSize="md-regular" />
                  <span className="capitalize">{t}</span>
                </button>
              );
            })}
          </div>
        </div>
        <FormInput
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="local api"
          autoFocus
        />
      </div>
    </DialogContainer>
  );
}

export function iconForType(type: ProjectItemType) {
  switch (type) {
    case "app":
      return Cube;
    case "database":
      return Database;
    case "queue":
      return Envelope;
    case "vault":
      return Lock;
  }
}

function itemHref(workspaceSlug: string, projectId: string, item: ProjectItem): string {
  const base = `/${workspaceSlug}/projects/${projectId}`;
  switch (item.type) {
    case "app":
      return `${base}/apps/${item.slug}`;
    case "database":
      return `${base}/databases/${item.slug}`;
    case "queue":
      return `${base}/queues/${item.slug}`;
    case "vault":
      return `${base}/vault/${item.slug}`;
  }
}
