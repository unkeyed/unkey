"use client";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/group-button";
import { RefreshCcw } from "lucide-react";
import { ONE_DAY_MS } from "../../constants";
import { useLogSearchParams } from "../../query-state";
import { DatePickerWithRange } from "./components/custom-date-filter";
import { ResponseStatus } from "./components/response-status";
import { SearchCombobox } from "./components/search-combobox/search-combobox";
import { Timeline } from "./components/timeline";

export const LogsFilters = () => {
  const { setSearchParams } = useLogSearchParams();

  const handleRefresh = () => {
    const now = Date.now();
    const startTime = now - ONE_DAY_MS;
    const endTime = Date.now();

    setSearchParams({
      endTime: endTime,
      host: null,
      method: null,
      path: null,
      requestId: null,
      responseStatus: [],
      startTime: startTime,
    });
  };

  return (
    <div className="relative mb-4">
      <div className="flex items-center gap-2 w-full flex-wrap">
        <div className="w-fit min-w-[330px]">
          <SearchCombobox />
        </div>

        <ButtonGroup>
          <Button variant="outline">
            <Timeline />
          </Button>
          <Button variant="outline">
            <DatePickerWithRange />
          </Button>
        </ButtonGroup>

        <Button variant="outline">
          <ResponseStatus />
        </Button>
        <Button variant="outline" size="icon" className="w-10" onClick={handleRefresh}>
          <RefreshCcw className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
};
