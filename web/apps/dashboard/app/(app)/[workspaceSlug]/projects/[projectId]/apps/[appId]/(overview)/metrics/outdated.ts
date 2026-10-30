import { cn } from "cn";

export function outdatedProps(outdated: boolean, className?: string) {
  return {
    className: cn("transition-opacity duration-150 ease-out", outdated && "opacity-35", className),
    inert: outdated,
  };
}
