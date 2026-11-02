"use client";

import { AnimatedColumn } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/animated-column";
import { Button } from "@unkey/ui";
import { ConfigCard } from "./config-card";
import { useNewAppFlow } from "./flow";
import { type Pane, cardView, columnCards } from "./wizard-model";

export function FlowColumn() {
  const { state } = useNewAppFlow();
  const { at, previous, current, next } = columnCards(state);
  return (
    <AnimatedColumn
      at={at}
      previous={previous}
      current={current}
      next={next}
      renderCard={(id) => <CardContent id={id} />}
    />
  );
}

function CardContent({ id }: { id: Pane }) {
  const { state, locked, dispatch } = useNewAppFlow();
  const view = cardView(state, id);
  if (!view) {
    return null;
  }
  const { previous } = view;
  return (
    <ConfigCard
      view={view}
      back={
        previous ? (
          <Button
            variant="outline"
            size="sm"
            disabled={locked}
            onClick={() => dispatch({ type: "go", step: previous })}
          >
            Back
          </Button>
        ) : null
      }
    />
  );
}
