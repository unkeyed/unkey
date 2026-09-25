"use client";
import {
  IconGridOutline18,
  IconMagnifierOutline18,
  IconSquareBulletListOutline18,
  IconXmarkOutline18,
} from "@unkey/icons";
import {
  Button,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  ResourceListHeader,
  Tabs,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import { type AppsView, parseAppsView } from "./use-apps-view";

type Props = {
  search: string;
  onSearchChange: (value: string) => void;
  view: AppsView;
  onViewChange: (view: AppsView) => void;
};

const VIEWS = [
  { view: "grid", label: "Grid view", Icon: IconGridOutline18 },
  { view: "list", label: "List view", Icon: IconSquareBulletListOutline18 },
] as const;

export function AppsListControls({ search, onSearchChange, view, onViewChange }: Props) {
  return (
    <ResourceListHeader className="items-center">
      <div className="flex h-8 w-full items-center md:w-80">
        <InputGroup className="h-8">
          <InputGroupAddon className="pointer-events-none">
            <IconMagnifierOutline18 className="size-4 text-gray-9" />
          </InputGroupAddon>
          <InputGroupInput
            aria-label="Search apps"
            type="text"
            value={search}
            maxLength={256}
            placeholder="Search by name, repo, image or domain"
            className="h-8 text-[13px] font-medium"
            onChange={(event) => onSearchChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                onSearchChange("");
              }
            }}
          />
          {search ? (
            <InputGroupAddon align="inline-end">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Clear search"
                onClick={() => onSearchChange("")}
              >
                <IconXmarkOutline18 className="size-4" />
              </Button>
            </InputGroupAddon>
          ) : null}
        </InputGroup>
      </div>
      <Tabs
        value={view}
        onValueChange={(value) => {
          const next = parseAppsView(value);
          if (next) {
            onViewChange(next);
          }
        }}
        className="md:ml-auto"
      >
        <TabsList aria-label="View" className="h-8 gap-0.5 border bg-raised p-0.5">
          {VIEWS.map(({ view: value, label, Icon }) => (
            <TabsTrigger
              key={value}
              value={value}
              aria-label={label}
              className="h-full w-7 px-0 py-0 text-gray-11 hover:text-gray-12 data-active:bg-grayA-3 data-active:text-gray-12 data-active:shadow-none"
            >
              <Icon className="size-3.5" />
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
    </ResourceListHeader>
  );
}
