"use client";

import { Plus, Trash } from "@unkey/icons";
import { Button, Input, SettingCard, toast } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import { type HeaderUpdateField, emptyHeader, headerUpdateFieldsSchema } from "../header-fields";
import { DrainShell } from "./drain-shell";
import type { DrainTelemetry, HttpDrain } from "./types";
import { useDrainUpdate } from "./use-drain-update";

type EditableHeader =
  | { id: number; source: "stored"; name: string; value: string }
  | { id: number; source: "new"; name: string; value: string };

export function HttpDrainDetail({ drain, ...telemetry }: { drain: HttpDrain } & DrainTelemetry) {
  const [destination, setDestination] = useState(drain.config.url);
  const nextHeaderId = useRef(Math.max(drain.config.headers.length, 1));
  const [headers, setHeaders] = useState<EditableHeader[]>(() =>
    editableHeaders(drain.config.headers, 0),
  );
  const update = useDrainUpdate(drain.id, (variables) => {
    if (
      variables.destination?.kind === "http" &&
      variables.destination.config.headers !== undefined
    ) {
      const fields = editableHeaders(
        variables.destination.config.headers.map((header) => header.name),
        nextHeaderId.current,
      );
      nextHeaderId.current += fields.length;
      setHeaders(fields);
    }
  });

  useEffect(() => setDestination(drain.config.url), [drain.config.url]);
  useEffect(() => {
    const fields = editableHeaders(drain.config.headers, nextHeaderId.current);
    nextHeaderId.current += fields.length;
    setHeaders(fields);
  }, [drain.config.headers]);

  const saveHeaders = () => {
    const requested: HeaderUpdateField[] = [];
    for (const header of headers) {
      const name = header.name.trim();
      if (header.source === "new" && name === "" && header.value === "") {
        continue;
      }
      if (header.source === "stored" && header.value === "") {
        requested.push({ mode: "preserve", name });
      } else {
        requested.push({ mode: "set", name, value: header.value });
      }
    }
    const parsed = headerUpdateFieldsSchema.safeParse(requested);
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? "Check the header fields");
      return;
    }
    update.mutate({
      id: drain.id,
      destination: {
        kind: "http",
        config: { headers: parsed.data },
      },
    });
  };

  const headersChanged =
    headers.filter((header) => header.source === "stored").length !== drain.config.headers.length ||
    headers.some((header) =>
      header.source === "stored"
        ? header.value !== ""
        : header.name.trim() !== "" || header.value !== "",
    );

  return (
    <DrainShell
      drain={drain}
      destination={destination}
      currentDestination={drain.config.url}
      onDestinationChange={setDestination}
      onSaveDestination={(url) =>
        update.mutate({ id: drain.id, destination: { kind: "http", config: { url } } })
      }
      update={update}
      {...telemetry}
    >
      <SettingCard
        title="Headers"
        description="Header values stay hidden after you save them. Leave a value blank to keep its current value."
        contentWidth="w-full lg:w-[520px] justify-end"
      >
        <div className="flex w-full flex-col gap-3">
          {headers.map((header, index) => (
            <div key={header.id} className="flex items-center gap-2">
              <Input
                aria-label={`Header ${index + 1} name`}
                placeholder="Header name"
                value={header.name}
                disabled={header.source === "stored"}
                onChange={(event) =>
                  setHeaders((current) =>
                    current.map((item, itemIndex) =>
                      itemIndex === index ? { ...item, name: event.target.value } : item,
                    ),
                  )
                }
              />
              <Input
                type="password"
                autoComplete="off"
                aria-label={`Header ${index + 1} value`}
                placeholder={header.source === "stored" ? "Enter a new value" : "Header value"}
                value={header.value}
                onChange={(event) =>
                  setHeaders((current) =>
                    current.map((item, itemIndex) =>
                      itemIndex === index ? { ...item, value: event.target.value } : item,
                    ),
                  )
                }
              />
              {header.source === "stored" || headers.length > 1 ? (
                <Button
                  variant="ghost"
                  size="sm"
                  className="size-9 shrink-0 justify-center px-0 text-gray-11"
                  aria-label={`Remove header ${index + 1}`}
                  onClick={() =>
                    setHeaders((current) => current.filter((_, itemIndex) => itemIndex !== index))
                  }
                >
                  <Trash iconSize="sm-regular" />
                </Button>
              ) : null}
            </div>
          ))}
          <div className="flex justify-between gap-2">
            <Button
              variant="outline"
              disabled={headers.length >= 32}
              onClick={() => {
                const id = nextHeaderId.current;
                nextHeaderId.current += 1;
                setHeaders((current) => [...current, { id, source: "new", ...emptyHeader }]);
              }}
            >
              <Plus iconSize="sm-regular" />
              Add header
            </Button>
            <Button
              variant="primary"
              loading={update.isLoading}
              disabled={!headersChanged}
              onClick={saveHeaders}
            >
              Save headers
            </Button>
          </div>
        </div>
      </SettingCard>
      <SettingCard
        title="Body format"
        description="JSON sends an array of events. NDJSON sends one event per line."
        contentWidth="w-full lg:w-[420px] justify-end"
      >
        <div className="flex rounded-lg border border-grayA-4 p-1">
          {(["json", "ndjson"] as const).map((value) => (
            <Button
              key={value}
              size="sm"
              variant={drain.config.format === value ? "primary" : "ghost"}
              loading={update.isLoading}
              onClick={() => {
                if (drain.config.format !== value) {
                  update.mutate({
                    id: drain.id,
                    destination: {
                      kind: "http",
                      config: { format: value },
                    },
                  });
                }
              }}
            >
              {value === "json" ? "JSON array" : "NDJSON"}
            </Button>
          ))}
        </div>
      </SettingCard>
    </DrainShell>
  );
}

function editableHeaders(headers: string[], firstId: number): EditableHeader[] {
  if (headers.length === 0) {
    return [{ id: firstId, source: "new", ...emptyHeader }];
  }
  return headers.map((name, index) => ({
    id: firstId + index,
    source: "stored",
    name,
    value: "",
  }));
}
