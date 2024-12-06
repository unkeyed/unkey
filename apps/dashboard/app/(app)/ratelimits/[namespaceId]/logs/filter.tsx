"use client";
import { ArrayInput } from "@/components/array-input";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  CalendarIcon,
  CalendarRange,
  CheckCheck,
  ChevronDown,
  RefreshCw,
  User,
  X,
} from "lucide-react";
import {
  parseAsArrayOf,
  parseAsBoolean,
  parseAsString,
  parseAsTimestamp,
  useQueryState,
} from "nuqs";
import type React from "react";
import { useState, useTransition } from "react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { format } from "date-fns";
import { useRouter } from "next/navigation";
import { Calendar } from "@/components/ui/calendar";
import { DateTimePicker } from "./datecomp";

export const Filters: React.FC = () => {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const [identifier, setIdentifier] = useQueryState(
    "identifier",
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    })
  );
  const [success, setSuccess] = useQueryState(
    "success",
    parseAsBoolean.withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    })
  );

  const [after, setAfter] = useQueryState(
    "after",
    parseAsTimestamp.withOptions({
      history: "push",
      shallow: false,
      clearOnDefault: true,
    })
  );

  const [before, setBefore] = useQueryState(
    "before",
    parseAsTimestamp.withOptions({
      history: "push",
      shallow: false,
      clearOnDefault: true,
    })
  );

  const [identifierVisible, setIdentifierVisible] = useState(false);
  const [successVisible, setSuccessVisible] = useState(false);
  const [timeRangeVisible, setTimeRangeVisible] = useState(false);

  return (
    <div className="flex flex-col w-full gap-2">
      <div className="flex items-center justify-end w-full gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="secondary" className="text-xs">
              Add Filter <ChevronDown className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem className="text-xs text-content-subtle">
              Filters are case sensitive
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => setIdentifierVisible(true)}>
              <User className="w-4 h-4 mr-2" />
              Identifier
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTimeRangeVisible(true)}>
              <CalendarRange className="w-4 h-4 mr-2" />
              Time
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setSuccess(true)}>
              <CheckCheck className="w-4 h-4 mr-2" />
              Success
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        {identifierVisible ? (
          <Button
            variant="outline"
            size="sm"
            className="flex items-center h-8 gap-2 bg-background-subtle"
            onClick={() => {
              startTransition(() => {
                setIdentifierVisible(false);
                setSuccessVisible(false);
                setSuccess(null);
                setTimeRangeVisible(false);
                setIdentifier(null);
                setAfter(null);
                setBefore(null);
              });
            }}
          >
            Reset Filters
            <X className="w-4 h-4" />
          </Button>
        ) : null}
        <Button
          size="icon"
          variant="secondary"
          onClick={() => {
            startTransition(router.refresh);
          }}
        >
          <RefreshCw className={cn("w-4 h-4", { "animate-spin": isPending })} />
        </Button>
      </div>
      <div className="flex flex-col items-start w-full gap-2">
        {identifierVisible || identifier.length > 0 ? (
          <FilterRow
            title="Identifiers"
            selected={identifier}
            setSelected={(v) => {
              setIdentifier(v);
              startTransition(() => {});
            }}
            removeFilter={() => setIdentifierVisible(false)}
          />
        ) : null}

        {successVisible || typeof success === "boolean" ? (
          <div className="flex items-center w-full gap-2">
            <Select
              value={success ? "true" : "false"}
              onValueChange={(v) => {
                startTransition(() => {
                  setSuccess(v === "true");
                });
              }}
            >
              <SelectTrigger>
                <SelectValue defaultValue={success ? "true" : "false"} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="true">Success</SelectItem>
                <SelectItem value="false">Ratelimited</SelectItem>
              </SelectContent>
            </Select>
            <Button
              size="icon"
              variant="secondary"
              onClick={() => {
                startTransition(() => {
                  setSuccessVisible(false);
                  setSuccess(null);
                });
              }}
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        ) : null}
        {timeRangeVisible || after !== null || before !== null ? (
          <div className="flex items-center w-full gap-2">
            <span className="mr-1 text-xs font-medium">From:</span>
            <DateTimePicker
              date={after ?? new Date()}
              setDate={(date) => setBefore(date)}
              timeInputLabel="Select Time"
              dateFormat="MM/dd/yyyy HH:mm"
              calendarProps={{
                disabled: { before: new Date() },
                showOutsideDays: true,
              }}
              timeInputProps={{
                className: "custom-time-input",
              }}
            >
              <Button>
                <CalendarIcon className="mr-2 h-4 w-4" />
                {after ? (
                  format(after, "yyyy-MM-dd'T'HH:mm")
                ) : (
                  <span>Pick a date</span>
                )}
              </Button>
            </DateTimePicker>
            <div className="flex items-center w-full h-8 p-1 text-sm border rounded-md group focus-within:border-primary">
              <div className="flex flex-wrap items-center w-full gap-1 px-2">
                <span className="mr-1 text-xs font-medium">Until:</span>
                <input
                  id="before"
                  type="datetime-local"
                  value={
                    before ? format(before, "yyyy-MM-dd'T'HH:mm") : undefined
                  }
                  onChange={(v) => {
                    startTransition(() => {
                      setBefore(new Date(v.currentTarget.value));
                    });
                  }}
                  className="flex-1 w-full bg-transparent outline-none placeholder:text-content-subtle text-xs font-medium [&::-webkit-calendar-picker-indicator]:text-xs"
                />
              </div>
            </div>
            <Button
              className="flex-shrink-0"
              size="icon"
              variant="secondary"
              onClick={() => {
                setTimeRangeVisible(false);
                setAfter(null);
                setBefore(null);
                startTransition(() => {});
              }}
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        ) : null}
      </div>
    </div>
  );
};

const FilterRow: React.FC<{
  title: string;
  selected: string[];
  setSelected: (v: string[]) => void;
  removeFilter: () => void;
}> = ({ title, selected, setSelected, removeFilter }) => {
  return (
    <div className="flex items-center w-full gap-2">
      <ArrayInput title={title} selected={selected} setSelected={setSelected} />
      <Button size="icon" variant="secondary" onClick={removeFilter}>
        <X className="w-4 h-4" />
      </Button>
    </div>
  );
};
