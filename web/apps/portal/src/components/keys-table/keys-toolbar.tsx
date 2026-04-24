import { Check, ChevronDown, Search } from "lucide-react";
import { Button } from "~/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "~/components/ui/dropdown-menu";
import { Input } from "~/components/ui/input";
import { cn } from "~/lib/utils";

export type StatusFilter = "all" | "enabled" | "disabled" | "expired";

const STATUS_LABELS: Record<StatusFilter, string> = {
  all: "All",
  enabled: "Enabled",
  disabled: "Disabled",
  expired: "Expired",
};

const STATUS_DOTS: Record<StatusFilter, string> = {
  all: "bg-gray-7",
  enabled: "bg-success-9",
  disabled: "bg-gray-9",
  expired: "bg-error-9",
};

type Props = {
  searchValue: string;
  onSearchChange: (value: string) => void;
  statusValue: StatusFilter;
  onStatusChange: (value: StatusFilter) => void;
};

export function KeysToolbar({
  searchValue,
  onSearchChange,
  statusValue,
  onStatusChange,
}: Props) {
  return (
    <div className="flex flex-1 items-center gap-2">
      <div className="relative max-w-sm flex-1">
        <Search className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-2.5 size-4 text-gray-10" />
        {/* TODO @davehawkins: debounce the search field before shipping to prod — today it fires on every keystroke */}
        <Input
          type="search"
          placeholder="Search by name, ID, or externalId"
          value={searchValue}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-8"
          aria-label="Search keys"
        />
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline">
            <StatusDot value={statusValue} />
            Status: {STATUS_LABELS[statusValue]}
            <ChevronDown />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="min-w-44">
          {(Object.keys(STATUS_LABELS) as StatusFilter[]).map((value) => (
            <DropdownMenuItem
              key={value}
              onSelect={(e) => {
                e.preventDefault();
                onStatusChange(value);
              }}
              className="justify-between"
            >
              <span className="flex items-center gap-2">
                <StatusDot value={value} />
                {STATUS_LABELS[value]}
              </span>
              {statusValue === value && <Check className="size-3.5 text-gray-11" />}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function StatusDot({ value }: { value: StatusFilter }) {
  return <span aria-hidden className={cn("size-2 shrink-0 rounded-full", STATUS_DOTS[value])} />;
}
