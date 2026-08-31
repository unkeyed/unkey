"use client";

import { Earth } from "@unkey/icons";
import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  useStepWizard,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { AxiomLogo } from "../axiom-logo";
import { DestinationStepContainer } from "./destination-step-container";
import type { Kind } from "./form-schema";

const OPTIONS: Array<{
  kind: Kind;
  title: string;
  description: string;
  icon: ReactNode;
}> = [
  {
    kind: "http",
    title: "HTTP",
    description: "Send audit logs to an HTTPS endpoint.",
    icon: <Earth className="size-[18px] text-gray-12" iconSize="md-medium" />,
  },
  {
    kind: "axiom",
    title: "Axiom",
    description: "Send audit logs to an Axiom dataset.",
    icon: <AxiomLogo className="size-[18px] text-gray-12" />,
  },
];

export function ChooseDestinationStep({ onSelect }: { onSelect: (kind: Kind) => void }) {
  const { next } = useStepWizard();

  const select = (kind: Kind) => {
    onSelect(kind);
    next();
  };

  return (
    <DestinationStepContainer>
      {OPTIONS.map((option) => (
        <Item key={option.kind} variant="outline" className="px-4 py-[18px]">
          <ItemMedia className="size-8 rounded-[10px] ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none">
            {option.icon}
          </ItemMedia>
          <ItemContent>
            <ItemTitle>{option.title}</ItemTitle>
            <ItemDescription>{option.description}</ItemDescription>
          </ItemContent>
          <ItemActions>
            <Button
              variant="outline"
              className="rounded-lg border-grayA-4 shadow-sm transition-[background-color,box-shadow] hover:bg-grayA-2 hover:shadow-md"
              onClick={() => select(option.kind)}
            >
              Use {option.title}
            </Button>
          </ItemActions>
        </Item>
      ))}
    </DestinationStepContainer>
  );
}
