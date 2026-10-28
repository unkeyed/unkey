"use client";

import {
  IconCheckOutline18,
  IconEyeOutline18,
  IconGaugeOutline18,
  IconPenWriting3Outline18,
  IconPlusOutline18,
  IconXmarkOutline18,
} from "@unkey/icons";
import { Button, Item, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";
import type { ReactNode } from "react";
import { useEditorCatalogue } from "./editor-catalogue";
import type { Policy } from "./lib/policy";
import type { TemplateId } from "./lib/templates";

export const TEMPLATE_ICONS: Record<TemplateId, ReactNode> = {
  read: <IconEyeOutline18 />,
  write: <IconPenWriting3Outline18 />,
  verify: <IconCheckOutline18 />,
  ratelimit: <IconGaugeOutline18 />,
  custom: <IconPlusOutline18 />,
};

type TemplateGalleryProps = {
  onPick: (policies: Policy[]) => void;
  onCancel?: () => void;
};

export function TemplateGallery({ onPick, onCancel }: TemplateGalleryProps) {
  const { templates } = useEditorCatalogue();
  return (
    <div className="flex flex-col gap-2">
      {onCancel ? (
        <div className="flex justify-end">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            aria-label="Close templates"
            className="size-8 shrink-0 justify-center rounded-lg px-0 text-gray-11 hover:bg-grayA-3 hover:text-gray-12"
            onClick={onCancel}
          >
            <IconXmarkOutline18 />
          </Button>
        </div>
      ) : null}
      <div className="grid gap-2 sm:grid-cols-2">
        {templates.map((template) => (
          <Item
            key={template.id}
            variant="outline"
            render={
              <button type="button" onClick={() => onPick(template.materialise())}>
                <ItemMedia>{TEMPLATE_ICONS[template.id]}</ItemMedia>
                <ItemContent>
                  <ItemTitle>{template.title}</ItemTitle>
                  <ItemDescription>{template.description}</ItemDescription>
                </ItemContent>
              </button>
            }
          />
        ))}
      </div>
    </div>
  );
}
