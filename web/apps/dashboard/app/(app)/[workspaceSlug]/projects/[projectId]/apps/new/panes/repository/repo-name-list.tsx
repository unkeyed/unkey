"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import { Github, IconCheckOutline12, IconMagnifierOutline12 } from "@unkey/icons";
import { Combobox, InputGroup, InputGroupAddon, InputGroupInput, Loading } from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { CardPending } from "../../card";
import { cardCopy } from "../../wizard-model";
import { type RepoItem, ownerOptions, resolveOwner, visibleRepos } from "./repository-view";

type RepoNameListProps = {
  repos: RepoItem[] | null;
  pending: { repoId: number | null; label: string } | null;
  linkedRepoId: number | null;
  disabled: boolean;
  onPick: (repo: RepoItem) => void;
};

type RowMark = "pending" | "linked" | "none";

const rowMark: Record<RowMark, (pendingLabel: string) => ReactNode> = {
  pending: (pendingLabel) => (
    <span className="flex shrink-0 items-center gap-2 text-xs font-normal text-gray-11">
      <Loading size={14} />
      {pendingLabel}
    </span>
  ),
  linked: () => <IconCheckOutline12 role="img" aria-label="Linked" className="text-gray-12" />,
  none: () => null,
};

function markFor(
  repoId: number,
  pendingRepoId: number | null,
  linkedRepoId: number | null,
): RowMark {
  if (pendingRepoId === repoId) {
    return "pending";
  }
  return repoId === linkedRepoId ? "linked" : "none";
}

function linkedFirst<T extends { repo: RepoItem }>(rows: T[], linkedRepoId: number | null): T[] {
  const linked = rows.filter(({ repo }) => repo.id === linkedRepoId);
  return [...linked, ...rows.filter(({ repo }) => repo.id !== linkedRepoId)];
}

function OwnerSelect({
  options,
  value,
  disabled,
  onSelect,
}: {
  options: { value: string; label: string }[];
  value: string;
  disabled: boolean;
  onSelect: (owner: string) => void;
}) {
  return (
    <Combobox
      aria-label="GitHub account"
      wrapperClassName="w-44 shrink-0"
      className="h-8 bg-transparent"
      leftIcon={<Github className="text-gray-12" />}
      options={options}
      value={value}
      onSelect={onSelect}
      disabled={disabled}
      placeholder={<span className="text-gray-9">Account</span>}
      searchPlaceholder="Filter accounts…"
      emptyMessage="No accounts match."
    />
  );
}

export function RepoNameList({
  repos,
  pending,
  linkedRepoId,
  disabled,
  onPick,
}: RepoNameListProps) {
  const [query, setQuery] = useState("");
  const [pickedOwner, setPickedOwner] = useState<string | null>(null);
  const owner = resolveOwner(repos ?? [], pickedOwner);
  const matches = visibleRepos(repos ?? [], owner, query);

  return (
    <div className="flex min-h-0 flex-col gap-3">
      <div className="flex shrink-0 gap-2">
        <OwnerSelect
          options={ownerOptions(repos ?? [])}
          value={owner}
          disabled={repos === null || repos.length === 0}
          onSelect={setPickedOwner}
        />
        <InputGroup className="h-8 bg-transparent">
          <InputGroupAddon>
            <IconMagnifierOutline12 className="shrink-0 text-gray-9" />
          </InputGroupAddon>
          <InputGroupInput
            autoFocus
            disabled={repos === null}
            data-1p-ignore
            autoComplete="off"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search repositories…"
            aria-label="Search repositories"
          />
        </InputGroup>
      </div>
      <div className="h-72 min-h-0 overflow-hidden rounded-lg border border-grayA-4">
        {repos === null ? (
          <CardPending message={cardCopy["pick-repo"].pending} className="h-full" />
        ) : (
          <ScrollArea className="h-full *:overscroll-contain">
            {matches.length === 0 ? (
              <p className="py-6 text-center text-sm text-gray-10">No repositories match.</p>
            ) : (
              <ul className="divide-y divide-grayA-4">
                {linkedFirst(matches, linkedRepoId).map(({ repo, label }) => (
                  <li key={repo.id}>
                    <button
                      type="button"
                      disabled={disabled}
                      aria-busy={pending?.repoId === repo.id}
                      onClick={() => onPick(repo)}
                      className="flex h-10 w-full items-center gap-3 px-4 text-left text-sm font-medium text-gray-12 cursor-pointer hover:bg-grayA-2 focus-visible:bg-grayA-2 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:not-aria-busy:opacity-50"
                    >
                      <span className="min-w-0 flex-1 truncate">{label}</span>
                      {rowMark[markFor(repo.id, pending?.repoId ?? null, linkedRepoId)](
                        pending?.label ?? "",
                      )}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </ScrollArea>
        )}
      </div>
    </div>
  );
}
