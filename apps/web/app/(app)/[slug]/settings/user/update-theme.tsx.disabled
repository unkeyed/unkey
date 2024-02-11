"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { LucideIcon, Monitor, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import React from "react";

export const UpdateTheme: React.FC = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Theme</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-8 max-sm:gap-2 ">
          <Option theme="light" icon={Sun} />
          <Option theme="dark" icon={Moon} />
          <Option theme="system" icon={Monitor} />
        </div>
      </CardContent>
    </Card>
  );
};

const Option: React.FC<{ theme: string; icon: LucideIcon }> = (props) => {
  const { theme, setTheme } = useTheme();
  return (
    <button
      type="button"
      onClick={() => setTheme(props.theme)}
      className={cn(
        "border text-sm rounded-md hover:border-primary flex items-center justify-center gap-2 h-8 p-2 ",
        {
          "bg-primary text-primary-foreground border-primary": props.theme === theme,
        },
      )}
    >
      <props.icon className="w-4 h-4 shrink-0" /> <span className="capitalize">{props.theme}</span>
    </button>
  );
};
