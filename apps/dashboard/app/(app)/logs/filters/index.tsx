import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/group-button";
import { Input } from "@/components/ui/input";
import { RefreshCcw, Search } from "lucide-react";
import { ChartsComp } from "../chart";
import { DatePickerWithRange } from "./components/custom-date-filter";
import { HourFilter } from "./components/hour-filter";
import { ResponseStatus } from "./components/response-status";
import React from "react";

export const LogsFilters = () => {
  return (
    <>
      <div className="flex items-center gap-2 w-full">
        <div className="w-[330px]">
          <Input type="text" placeholder="Search logs" startIcon={Search} />
        </div>
        <Button variant="outline" size="icon" className="w-10">
          <RefreshCcw className="h-4 w-4" />
        </Button>
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
      </div>
      <ChartsComp />
    </>
  );
};
