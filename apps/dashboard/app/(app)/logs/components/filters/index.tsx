"use client";
import { ButtonGroup } from "@/components/ui/group-button";
import { Button } from "@unkey/ui";
import { RefreshCcw } from "lucide-react";
import { useLogSearchParams } from "../../query-state";
import { DatePickerWithRange } from "./components/custom-date-filter";
import { ResponseStatus } from "./components/response-status";
import { SearchCombobox } from "./components/search-combobox/search-combobox";
import { Timeline } from "./components/timeline";

export const LogsFilters = () => {
  const { setSearchParams } = useLogSearchParams();

  const handleRefresh = () => {
    setSearchParams({
      endTime: null,
      host: null,
      method: null,
      path: null,
      requestId: null,
      responseStatus: [],
      startTime: null,
    });
  };

  return (
    <div className="relative mb-4">
      <div className="flex items-center gap-2 w-full flex-wrap">
        <SearchCombobox />

        <ButtonGroup>
          <Button>
            <Timeline />
          </Button>
          <Button>
            <DatePickerWithRange />
          </Button>
        </ButtonGroup>

        <Button>
          <ResponseStatus />
        </Button>
        <Button shape="square" className="w-10" onClick={handleRefresh}>
          <RefreshCcw className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
};
