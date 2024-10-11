"use client";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/group-button";
import { RefreshCcw } from "lucide-react";
import { DatePickerWithRange } from "./components/custom-date-filter";
import { HourFilter } from "./components/hour-filter";
import { ResponseStatus } from "./components/response-status";
import { SearchCombobox } from "./components/search-combobox/search-combobox";

export const LogsFilters = () => {
  return (
    <div className="relative mb-4">
      <div className="flex items-center gap-2 w-full flex-wrap">
        <div className="w-fit min-w-[330px]">
          <SearchCombobox />
        </div>

        <ButtonGroup>
          <Button variant="outline">
            <HourFilter />
          </Button>
          <Button variant="outline">
            <DatePickerWithRange />
          </Button>
        </ButtonGroup>

        <Button variant="outline">
          <ResponseStatus />
        </Button>
        <Button variant="outline" size="icon" className="w-10">
          <RefreshCcw className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
};
