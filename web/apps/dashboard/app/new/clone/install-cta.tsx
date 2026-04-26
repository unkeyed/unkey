"use client";

import { Github, Layers3 } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import type { ParsedRepository } from "./parse-params";

type InstallCtaProps = {
  repository: ParsedRepository;
  cloneSearch: string;
};

export function InstallCta({ repository, cloneSearch }: InstallCtaProps) {
  const state = JSON.stringify({ returnTo: "clone", params: cloneSearch });
  const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;

  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-xl flex flex-col items-center">
        <Empty>
          <Empty.Icon>
            <Layers3 className="size-7 text-accent-12" iconSize="xl-medium" />
          </Empty.Icon>
          <Empty.Title>Install the Unkey GitHub App</Empty.Title>
          <Empty.Description>
            We need access to{" "}
            <a
              href={repository.url}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium text-gray-12 underline underline-offset-2"
            >
              {repository.fullName}
            </a>{" "}
            before we can deploy it. Install the Unkey GitHub App on this repository, then you'll
            come right back here.
          </Empty.Description>
          <Empty.Actions>
            <a href={installUrl}>
              <Button variant="primary" size="lg" className="rounded-lg">
                <Github className="size-[18px]! shrink-0" />
                <span className="text-[13px] font-medium">Install on GitHub</span>
              </Button>
            </a>
          </Empty.Actions>
        </Empty>
      </div>
    </div>
  );
}
