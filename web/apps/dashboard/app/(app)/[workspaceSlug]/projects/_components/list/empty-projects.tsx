import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ArrowRight, BookBookmark, Code, Cube, Earth, Github, HeartPulse } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import type { ReactNode } from "react";

type IconBoxProps = {
  children?: ReactNode;
  large?: boolean;
  className?: string;
};

const IconBox = ({ children, large, className }: IconBoxProps) => (
  <div
    className={cn(
      "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none",
      large ? "size-16" : "size-9",
      className,
    )}
  >
    {children}
  </div>
);

// The project box (center, focal) with repo (GitHub) and build (code) as its immediate flanks, then
// regions (globe) and liveness (pulse) on the outer edges. Boxes dim toward the edges and the row
// fades out at both ends so the box stays the focus.
const flankItems: { icon: ReactNode; large?: boolean; opacity: string }[] = [
  { icon: <Earth className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-50" },
  { icon: <Github className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Cube className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <Code className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-50" },
];

const ProjectIconRow = () => (
  <div
    className="p-2 mb-8"
    style={{
      maskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
    }}
  >
    <div className="flex gap-6 items-center justify-center text-gray-12">
      {flankItems.map((item, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static row, index is stable
        <IconBox key={i} large={item.large} className={item.opacity}>
          {item.icon}
        </IconBox>
      ))}
    </div>
  </div>
);

export function EmptyProjects() {
  const workspace = useWorkspaceNavigation();

  return (
    <div className="grow w-full flex justify-center items-center p-12">
      <div className="flex flex-col items-center text-center">
        <ProjectIconRow />

        <h2 className="text-accent-12 font-semibold text-2xl leading-8 mb-1">Projects</h2>
        <p className="text-accent-11 text-sm leading-6 max-w-md text-balance mb-6">
          Build, deploy and scale your API inside Unkey. Create a project to get started,
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
  );
}
