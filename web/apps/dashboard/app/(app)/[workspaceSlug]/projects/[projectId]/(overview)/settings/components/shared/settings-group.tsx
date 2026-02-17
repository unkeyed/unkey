"use client";

import { ChevronRight } from "@unkey/icons";
import React, { useState } from "react";

type SettingsGroupProps = {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
  defaultExpanded?: boolean;
};

export const SettingsGroup = ({
  icon,
  title,
  children,
  defaultExpanded = true,
}: SettingsGroupProps) => {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between mb-4 px-2">
        <div className="flex items-center gap-2.5">
          <div className="text-gray-9">{icon}</div>
          <span className="font-medium text-gray-12 text-[13px] leading-4">{title}</span>
        </div>
        <button
          type="button"
          onClick={() => setExpanded((prev) => !prev)}
          className="flex items-center gap-1 text-xs text-gray-10 hover:text-gray-11 transition-colors group duration-300"
        >
          {expanded ? "Hide" : "Show"}
          <ChevronRight
            className="text-gray-10 group-hover:text-gray-11 transition-all duration-300 flex-shrink-0"
            iconSize="sm-medium"
            style={{
              transitionTimingFunction: "cubic-bezier(.62,.16,.13,1.01)",
              transform: expanded ? "rotate(270deg)" : "rotate(90deg)",
            }}
          />
        </button>
      </div>
      <div
        className="grid transition-[grid-template-rows] duration-300"
        style={{
          gridTemplateRows: expanded ? "1fr" : "0fr",
          transitionTimingFunction: "cubic-bezier(.62,.16,.13,1.01)",
        }}
      >
        <div className="overflow-hidden">
          {React.Children.map(children, (child, index) => (
            <div
              className="transition-all duration-300"
              style={{
                transitionTimingFunction: "cubic-bezier(.62,.16,.13,1.01)",
                transitionDelay: expanded ? `${index * 50}ms` : "0ms",
                opacity: expanded ? 1 : 0,
                transform: expanded ? "translateY(0)" : "translateY(-8px)",
              }}
            >
              {child}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
