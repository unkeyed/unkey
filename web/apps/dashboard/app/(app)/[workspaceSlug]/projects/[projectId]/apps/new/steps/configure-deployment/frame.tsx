"use client";

import { PageHeaderBack, PageHeaderContent, PageHeaderTitle, useStepWizard } from "@unkey/ui";
import type { ReactNode } from "react";

export function ConfigureFrame({ children }: { children: ReactNode }) {
  const { back } = useStepWizard();
  return (
    <div className="mx-auto flex w-full max-w-225 flex-col gap-8 px-6 pt-8 pb-16">
      <PageHeaderContent className="items-start gap-1">
        <PageHeaderBack render={<button type="button" onClick={back} />} className="mb-2">
          Back
        </PageHeaderBack>
        <PageHeaderTitle>Configure deployment</PageHeaderTitle>
      </PageHeaderContent>
      {children}
    </div>
  );
}
