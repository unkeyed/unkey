"use client";

import { Button } from "@unkey/ui";
import {
  COLUMN_CARD_MAX_HEIGHT,
  COLUMN_GAP,
  COLUMN_TOP_SPACE,
  CardTitle,
  cardFooter,
} from "./card";
import { RepoListPlaceholder } from "./panes/repository/repo-name-list";
import { cardCopy } from "./wizard-model";

const copy = cardCopy["pick-repo"];

// Display only: the server verifies the signed state. Returns the project a
// GitHub install returns to when it was started from the new-app flow.
export function newAppReturnProject(state: string | null): string | null {
  if (!state) {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(state);
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      !("flow" in parsed) ||
      parsed.flow !== "app" ||
      ("returnTo" in parsed && parsed.returnTo === "settings") ||
      !("projectId" in parsed) ||
      typeof parsed.projectId !== "string"
    ) {
      return null;
    }
    return parsed.projectId;
  } catch {
    return null;
  }
}

export function GithubConnectingCard() {
  return (
    <div className="flex w-full flex-1 flex-col">
      <div
        className="grid grid-cols-[minmax(0,560px)] justify-center px-8"
        style={{ rowGap: COLUMN_GAP, gridTemplateRows: `${COLUMN_TOP_SPACE} auto` }}
      >
        <div
          className="row-start-2 flex min-h-0 flex-col overflow-hidden rounded-lg border bg-raised"
          style={{ maxHeight: COLUMN_CARD_MAX_HEIGHT }}
        >
          <div className="flex min-h-0 flex-col gap-5 p-5">
            <CardTitle title={copy.title} description={copy.description} />
            <RepoListPlaceholder message="Finalizing GitHub connection…" />
          </div>
          <div className={cardFooter}>
            <Button variant="outline" size="sm" disabled>
              Back
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
