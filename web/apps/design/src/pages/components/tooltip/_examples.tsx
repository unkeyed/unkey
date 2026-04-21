import {
  Button,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  return (
    <Preview>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline">Rotate key</Button>
          </TooltipTrigger>
          <TooltipContent>
            Generates a new secret and invalidates the old one.
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </Preview>
  );
}

export function PositioningExample() {
  return (
    <Preview>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline">Top</Button>
          </TooltipTrigger>
          <TooltipContent side="top">Anchored to the top edge</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline">Right</Button>
          </TooltipTrigger>
          <TooltipContent side="right">
            Anchored to the right edge
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline">Bottom, end-aligned</Button>
          </TooltipTrigger>
          <TooltipContent side="bottom" align="end" sideOffset={8}>
            Offset by 8px, aligned to the trigger's end
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </Preview>
  );
}

export function DelayExample() {
  return (
    <Preview>
      <TooltipProvider delayDuration={0}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline">Instant</Button>
          </TooltipTrigger>
          <TooltipContent>Opens immediately on hover.</TooltipContent>
        </Tooltip>
        <Tooltip delayDuration={700}>
          <TooltipTrigger asChild>
            <Button variant="outline">Patient</Button>
          </TooltipTrigger>
          <TooltipContent>Waits 700ms before opening.</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </Preview>
  );
}
