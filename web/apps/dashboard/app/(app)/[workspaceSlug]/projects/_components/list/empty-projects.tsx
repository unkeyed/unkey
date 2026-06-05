import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ArrowRight, BookBookmark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { ClassicProjectIcon } from "../classic-icon-row";
import { ArtStyleSwitcher, useArtStyle } from "../use-art-style";
import { UnkeyAscii } from "./unkey-ascii";

export function EmptyProjects() {
  const workspace = useWorkspaceNavigation();
  const [style] = useArtStyle();

  return (
    <div className="grow flex items-center justify-center p-12">
      <ArtStyleSwitcher />
      <div className="w-full max-w-4xl flex flex-col items-center">
        <div className="flex flex-col items-center text-center">
          {style === "ascii" ? (
            <UnkeyAscii className="mb-8" />
          ) : (
            <ClassicProjectIcon className="mb-8" />
          )}

          <h2 className="text-accent-12 font-semibold text-2xl leading-8 mb-1">Projects</h2>
          <p className="text-accent-11 text-sm leading-6 max-w-md text-balance mb-6">
            Build, deploy and scale your API straight from GitHub. Create a project to get started,
            free during beta.
          </p>

          <div className="flex items-center justify-center gap-3">
            <Link href={`/${workspace.slug}/projects/new`}>
              <Button variant="primary" size="md">
                Create your first project
                <ArrowRight />
              </Button>
            </Link>
            <a
              href="https://www.unkey.com/docs/quickstart/docs"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button variant="outline" size="md">
                <BookBookmark />
                Read the docs
              </Button>
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
