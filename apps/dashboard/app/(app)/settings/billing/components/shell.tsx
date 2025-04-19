"use client";

import { PageContent } from "@/components/page-content";
import type { PropsWithChildren } from "react";

export const Shell: React.FC<PropsWithChildren<{ workspace: { id: string; name: string } }>> = ({
  children,
}) => {
  return (
    <div>
      <PageContent>
        <div className="flex items-center justify-center w-full py-3 ">
          <div className="w-[760px] mt-4 flex flex-col justify-center items-center gap-5">
            <h1 className="w-full py-6 text-lg font-semibold text-left border-b text-accent-12 border-gray-4">
              Billing Settings
            </h1>
            {children}
          </div>
        </div>
      </PageContent>
    </div>
  );
};
