"use client";

import { Eye, Gauge, PenWriting3, ShieldCheck, Sliders, XMark } from "@unkey/icons";
import { Button, Item, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";
import type { ReactNode } from "react";
import type { Policy } from "../lib/policy";
import { TEMPLATES, type TemplateId } from "../lib/templates";

const TEMPLATE_ICONS: Record<TemplateId, ReactNode> = {
  read: <Eye iconSize="sm-regular" />,
  write: <PenWriting3 iconSize="sm-regular" />,
  verify: <ShieldCheck iconSize="sm-regular" />,
  ratelimit: <Gauge iconSize="sm-regular" />,
  custom: <Sliders iconSize="sm-regular" />,
};

type TemplateGalleryProps = {
  onPick: (policies: Policy[]) => void;
  onCancel?: () => void;
};

export function TemplateGallery({ onPick, onCancel }: TemplateGalleryProps) {
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
            <XMark iconSize="sm-regular" />
          </Button>
        </div>
      ) : null}
      <div className="grid gap-2 sm:grid-cols-2">
        {TEMPLATES.map((template) => (
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
