"use client";

import { IconChevronLeftOutline12 } from "@unkey/icons";
import { Button, PageHeaderContent, PageHeaderTitle, useStepWizard } from "@unkey/ui";
import type { ReactNode } from "react";

export function ConfigureFrame({ children }: { children: ReactNode }) {
  const { back } = useStepWizard();
  return (
    <div className="mx-auto flex w-full max-w-225 flex-col gap-8 px-6 pt-8 pb-16">
      <PageHeaderContent className="items-start gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={back}
          className="-ml-2 mb-2 gap-1 text-gray-10 hover:text-gray-12"
        >
          <IconChevronLeftOutline12 />
          Back
        </Button>
        <PageHeaderTitle>Configure deployment</PageHeaderTitle>
      </PageHeaderContent>
      {children}
    </div>
  );
}
