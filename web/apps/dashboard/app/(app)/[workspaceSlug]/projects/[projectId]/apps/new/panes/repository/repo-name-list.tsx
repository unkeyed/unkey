"use client";

import { IconMagnifierOutline12 } from "@unkey/icons";
import { InputGroup, InputGroupAddon, InputGroupInput, Loading } from "@unkey/ui";
import { useState } from "react";
import { type RepoItem, repoShortName } from "./repository-view";

type RepoNameListProps = {
  repos: RepoItem[];
  pendingRepoId: number | null;
  onPick: (repo: RepoItem) => void;
};

export function RepoListPlaceholder({ message }: { message: string }) {
  return (
    <div className="flex min-h-0 flex-col gap-3">
      <InputGroup className="h-8 shrink-0 bg-transparent" data-disabled>
        <InputGroupAddon>
          <IconMagnifierOutline12 className="shrink-0 text-gray-9" />
        </InputGroupAddon>
        <InputGroupInput
          disabled
          placeholder="Search repositories…"
          aria-label="Search repositories"
        />
      </InputGroup>
      <div className="flex h-[390px] min-h-24 shrink items-center justify-center gap-2 rounded-lg border border-grayA-4 text-sm text-gray-10">
        <Loading size={14} />
        <span aria-live="polite">{message}</span>
      </div>
      <div aria-hidden className="h-4 shrink-0" />
    </div>
  );
}

export function RepoNameList({ repos, pendingRepoId, onPick }: RepoNameListProps) {
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();
  const matches = needle
    ? repos.filter((repo) => repo.fullName.toLowerCase().includes(needle))
    : repos;

  return (
    <div className="flex min-h-0 flex-col gap-3">
      <InputGroup className="h-8 shrink-0 bg-transparent">
        <InputGroupAddon>
          <IconMagnifierOutline12 className="shrink-0 text-gray-9" />
        </InputGroupAddon>
        <InputGroupInput
          autoFocus
          data-1p-ignore
          autoComplete="off"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search repositories…"
          aria-label="Search repositories"
        />
      </InputGroup>
      <div className="flex min-h-0 flex-col overflow-hidden rounded-lg border border-grayA-4">
        <div className="min-h-0 overflow-y-auto overscroll-contain [scrollbar-color:var(--color-grayA-6)_transparent] [scrollbar-width:thin]">
          {matches.length === 0 ? (
            <p className="py-6 text-center text-sm text-gray-10">No repositories match.</p>
          ) : (
            <ul className="divide-y divide-grayA-4">
              {matches.map((repo) => (
                <li key={repo.id}>
                  <button
                    type="button"
                    title={repo.fullName}
                    disabled={pendingRepoId !== null}
                    onClick={() => onPick(repo)}
                    className="flex h-10 w-full items-center gap-3 px-4 text-left text-sm text-gray-12 hover:bg-grayA-2 focus-visible:bg-grayA-2 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <span className="min-w-0 flex-1 truncate">{repoShortName(repo.fullName)}</span>
                    {pendingRepoId === repo.id ? <Loading size={14} /> : null}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
