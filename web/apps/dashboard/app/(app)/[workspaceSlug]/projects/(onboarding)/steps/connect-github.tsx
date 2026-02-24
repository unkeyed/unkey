"use client";
import { Github, Layers3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { OnboardingLinks } from "./onboarding-links";

export const ConnectGithubStep = () => {
  return (
    <div className="flex flex-col items-center">
      <div className="border border-grayA-5 rounded-[14px] flex justify-center items-center gap-4 py-[18px] px-4 min-w-[600px]">
        <div className="size-8 rounded-[10px] bg-gray-12 grid place-items-center">
          <Layers3 className="size-[18px] text-gray-1" iconSize="md-medium" />
        </div>
        <div className="flex flex-col gap-3">
          <span className="font-medium text-gray-12 text-[13px] leading-[9px]">Import project</span>
          <span className="text-gray-10 text-[13px] leading-[9px]">
            Add a repo from your GitHub account
          </span>
        </div>
        <Button
          variant="outline"
          className="ml-20 rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
        >
          <Github className="!size-[18px] text-gray-12 shrink-0" />
          <a
            href={""}
            className="text-sm text-gray-12 font-medium"
            target="_blank"
            rel="noopener noreferrer"
          >
            Import from GitHub
          </a>
        </Button>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
