"use client";

import * as React from "react";
import { Command as CommandPrimitive } from "cmdk";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { useRouter } from "next/navigation";
import type { Glossary } from "@/.content-collections/generated";

export function FilterableCommand(props: {
  placeholder: string;
  className?: string;
  terms: Array<Glossary>;
}) {
  const [open, setOpen] = React.useState(false);
  const router = useRouter();
  const commandRef = React.useRef<HTMLDivElement>(null);

  return (
    <Command className={cn("h-auto [&>div]:border-b-0", props.className)} ref={commandRef}>
      <CommandInput
        placeholder={props.placeholder}
        onFocus={() => setOpen(true)}
        // The `onBlur` event checks if the new focus target (relatedTarget) is within the Command component:
        // - If it's not (i.e., clicking outside), it closes the list.
        // - If it is (i.e., selecting an item), it keeps the list open, allowing the `onSelect` to handle the navigation.
        onBlur={(event: React.FocusEvent<HTMLInputElement>) => {
          const relatedTarget = event.relatedTarget as Node | null;
          if (!commandRef.current?.contains(relatedTarget)) {
            setOpen(false);
          }
        }}
      />

      {open && (
        <CommandList>
          <CommandEmpty>No terms found.</CommandEmpty>
          <CommandGroup heading="Glossary Terms">
            {props.terms.map((item) => (
              <CommandPrimitive.Item
                key={item.slug}
                value={item.slug}
                className="px-3 py-2 cursor-pointer flex items-center w-full text-sm text-white/60 hover:text-white"
                onSelect={() => router.push(`/glossary/${item.slug}`)}
              >
                {item.title}
              </CommandPrimitive.Item>
            ))}
          </CommandGroup>
        </CommandList>
      )}
    </Command>
  );
}
