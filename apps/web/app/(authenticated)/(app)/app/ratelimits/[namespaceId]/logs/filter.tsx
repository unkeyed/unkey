"use client";
import { ArrayInput } from "@/components/array-input";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  CalendarRange,
  CheckCheck,
  ChevronDown,
  Earth,
  Locate,
  RefreshCw,
  User,
  X,
} from "lucide-react";
import {
  parseAsArrayOf,
  parseAsBoolean,
  parseAsIsoDateTime,
  parseAsString,
  useQueryState,
} from "nuqs";
import React, { useEffect, useState, useTransition } from "react";

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
import { useRouter } from "next/navigation";

export const Filters: React.FC = () => {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const [identifier, setIdentifier] = useQueryState(
    "identifier",
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  const [country, setCountry] = useQueryState(
    "country",
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  const [ipAddress, setIpAddress] = useQueryState(
    "ipAddress",
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  const [success, setSuccess] = useQueryState(
    "success",
    parseAsBoolean.withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  const [after, setAfter] = useQueryState(
    "after",
    parseAsIsoDateTime.withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  const [before, setBefore] = useQueryState(
    "before",
    parseAsIsoDateTime.withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

  const [localTime, setLocalTime] = useState("");
  useEffect(() => {
    setLocalTime(after?.toLocaleString() ?? "");
  }, [after]);

  const [identifierVisible, setIdentifierVisible] = useState(false);
  const [ipAddressVisible, setIpAddressVisible] = useState(false);
  const [countryVisible, setCountryVisible] = useState(false);
  const [successVisible, setSuccessVisible] = useState(false);
  const [timeRangeVisible, setTimeRangeVisible] = useState(false);
  return (
    <div className="flex flex-col w-full gap-2">
      <div className="flex items-center justify-end w-full gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger>
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
            <DropdownMenuItem onClick={() => setIpAddressVisible(true)}>
              <Locate className="w-4 h-4 mr-2" />
              IP address
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setCountryVisible(true)}>
              <Earth className="w-4 h-4 mr-2" />
              Country
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
        {identifierVisible || ipAddressVisible || countryVisible ? (
          <Button
            variant="outline"
            size="sm"
            className="flex items-center h-8 gap-2 bg-background-subtle"
            onClick={() => {
              setIdentifierVisible(false);
              setIdentifier(null);
              setIpAddressVisible(false);
              setIpAddress(null);
              setCountryVisible(false);
              setCountry(null);
              startTransition(() => {});
            }}
          >
            Clear
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
        {countryVisible || country.length > 0 ? (
          <FilterRow
            title="Countries"
            selected={country}
            setSelected={(v) => {
              setCountry(v);
              startTransition(() => {});
            }}
            removeFilter={() => setCountryVisible(false)}
          />
        ) : null}
        {ipAddressVisible || ipAddress.length > 0 ? (
          <FilterRow
            title="IP address"
            selected={ipAddress}
            setSelected={(v) => {
              setIpAddress(v);
              startTransition(() => {});
            }}
            removeFilter={() => setIpAddressVisible(false)}
          />
        ) : null}

        {successVisible || typeof success === "boolean" ? (
          <div className="flex items-center w-full gap-2">
            <Select
              value={success ? "true" : "false"}
              onValueChange={(v) => {
                setSuccess(v === "true");
                startTransition(() => {});
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
                setSuccessVisible(false);
                setSuccess(null);
                startTransition(() => {});
              }}
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        ) : null}
        {timeRangeVisible || after !== null || before !== null ? (
          <div className="flex items-center w-full gap-2">
            <div className="flex items-center w-full h-8 p-1 text-sm border rounded-md group focus-within:border-primary">
              <div className="flex flex-wrap items-center w-full gap-1 px-2">
                <span className="mr-1 text-xs font-medium">From:</span>
                --{after?.toLocaleString()}-- --{localTime}--
                <input
                  type="datetime-local"
                  value={after?.toLocaleString()}
                  onChange={(v) => {
                    setAfter(new Date(v.currentTarget.value));
                    startTransition(() => {});
                  }}
                  className="flex-1 w-full bg-transparent outline-none placeholder:text-content-subtle"
                />
              </div>
            </div>
            <div className="flex items-center w-full h-8 p-1 text-sm border rounded-md group focus-within:border-primary">
              <div className="flex flex-wrap items-center w-full gap-1 px-2">
                <span className="mr-1 text-xs font-medium">Until:</span>
                <input
                  id="before"
                  type="datetime-local"
                  value={before?.toLocaleString()}
                  onChange={(v) => {
                    setBefore(new Date(v.currentTarget.value));
                    startTransition(() => {});
                  }}
                  className="flex-1 w-full bg-transparent outline-none placeholder:text-content-subtle"
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
