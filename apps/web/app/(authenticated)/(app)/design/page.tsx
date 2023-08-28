"use client";
import { cn } from "@/lib/utils";
import { useTheme } from "next-themes";
import React, { PropsWithChildren } from "react";

import { Switch } from "@/components/ui/switch";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Label } from "@/components/ui/label";
import { Moon, Sun } from "lucide-react";
export default function Page() {
  const { theme, setTheme } = useTheme();

  return (
    <main className="max-w-2xl px-4 py-24 mx-auto space-y-16 sm:px-6 lg:max-w-7xl lg:px-8">
      <div className="flex items-center justify-between w-full">
        <h1 className="font-bold tracking-tight text-content text-7xl">Design</h1>
        <div className="flex items-center justify-between gap-2">
          <Label>{theme === "light" ? <Sun /> : <Moon />}</Label>
          <Switch
            checked={theme === "light"}
            onCheckedChange={(checked) => {
              console.log({ checked });
              setTheme(checked ? "light" : "dark");
            }}
          />
        </div>
      </div>

      <Section title="Black / White">
        <Grid>
          <Swatch name="black" bgColor="bg-black" />
          <Swatch name="white" bgColor="bg-white" />
        </Grid>
      </Section>

      <Section title="Background Colors" description="These are the background colors.">
        <Grid>
          <Swatch name="background" bgColor="bg-background" />
          <Swatch name="subtle" bgColor="bg-background-subtle" />
          <Swatch name="warn" bgColor="bg-warn" />
          <Swatch name="alert" bgColor="bg-alert" />
        </Grid>
      </Section>

      <Section title="Brand Colors">
        <Grid>
          <Swatch name="brand" bgColor="bg-brand" />
          <Swatch name="brand-foreground" bgColor="bg-brand-foreground" />
        </Grid>
      </Section>
      <Section title="Text Colors">
        <Grid>
          <Swatch textColor="text-content" name="content" bgColor="bg-content" />
          <Swatch textColor="text-content-subtle" name="content-subtle" bgColor="bg-content-subtle" />
          <Swatch textColor="text-content-info" name="content-info" bgColor="bg-content-info" />
          <Swatch textColor="text-content-warn" name="content-warn" bgColor="bg-content-warn" />
          <Swatch textColor="text-content-alert" name="content-alert" bgColor="bg-content-alert" />
        </Grid>
      </Section>

      <Section title="Text On Background">
        <Grid>
          <SwatchWithText
            borderColor="border-border"
            textColor="text-content"
            name="base"
            bgColor="bg-background"
          />
          <SwatchWithText
            borderColor="border-subtle"
            textColor="text-subtle-foreground"
            name="subtle"
            bgColor="bg-subtle"
          />
          <SwatchWithText
            borderColor="border-info"
            textColor="text-info-foreground"
            name="info"
            bgColor="bg-info"
          />
          <SwatchWithText
            borderColor="border-secondary"
            textColor="text-secondary-foreground"
            name="secondary"
            bgColor="bg-secondary"
          />
          <SwatchWithText
            borderColor="border-warn"
            textColor="text-warn-foreground"
            name="warn"
            bgColor="bg-warn"
          />
          <SwatchWithText
            borderColor="border-alert"
            textColor="text-alert-foreground"
            name="alert"
            bgColor="bg-alert"
          />
          <SwatchWithText
            borderColor="border-brand"
            textColor="text-brand-foreground"
            name="brand"
            bgColor="bg-brand"
          />
        </Grid>
      </Section>

      <Section title="Border / Ring">
        <Grid>
          <Swatch name="border" bgColor="bg-border" />
          <Swatch name="border-subtle" bgColor="bg-border-subtle" />
        </Grid>
      </Section>
    </main>
  );
}

const Section: React.FC<PropsWithChildren<{ title: string; description?: string }>> = ({
  children,
  title,
  description,
}) => (
  <div>
    <div className="max-w-xl">
      <h1 id="order-history-heading" className="text-3xl font-bold tracking-tight text-content">
        {title}
      </h1>
      <p className="mt-2 text-sm text-gray-500">{description}</p>
    </div>
    {children}
  </div>
);

const Grid: React.FC<PropsWithChildren> = ({ children }) => (
  <div className="grid grid-cols-1 gap-x-6 gap-y-10 sm:grid-cols-2 sm:gap-y-16 lg:grid-cols-3 lg:gap-x-8 xl:grid-cols-4">
    {children}
  </div>
);

const Swatch: React.FC<{
  bgColor: string;
  name: string;
  textColor?: string;
  borderColor?: string;
}> = ({ bgColor, name, textColor, borderColor }) => (
  <div className="relative flex items-center justify-start gap-4 p-4 duration-500 rounded-lg group hover:bg-background-subtle">
    <div
      className={cn(
        "w-16 h-16 rounded-lg border  shadow-lg shrink-0",
        bgColor,
        borderColor ?? "border-border",
      )}
    />
    <h3 className={cn("w-full font-mono text-sm ", textColor)}>{name}</h3>
    <CopyButton value={bgColor} />
  </div>
);

const SwatchWithText: React.FC<{
  bgColor: string;
  name: string;
  textColor: string;
  borderColor?: string;
}> = ({ bgColor, name, textColor, borderColor }) => (
  <div className="relative flex items-center justify-start gap-4 p-4 duration-500 rounded-lg group hover:bg-background-subtle">
    <div
      className={cn(
        "w-full h-16 rounded-lg border border-border shadow-lg shrink-0 flex items-center justify-center p-4",
        bgColor,
        borderColor ?? "border-border",
      )}
    >
      <h3 className={cn("font-mono text-sm ", textColor)}>{name}</h3>
    </div>
  </div>
);
