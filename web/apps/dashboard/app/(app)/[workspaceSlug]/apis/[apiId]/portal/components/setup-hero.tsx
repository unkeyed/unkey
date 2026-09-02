"use client";

import { Button, EmptyHero } from "@unkey/ui";
import {
  IconEarthOutline18,
  IconKeyOutline18,
  IconShieldKeyOutline18,
  IconUserOutline18,
  IconWindowLayoutOutline18,
} from "nucleo-ui-outline-18";

export function SetupHero({ onEnable }: { onEnable: () => void }) {
  return (
    <div className="flex w-full justify-center rounded-lg border border-grayA-4 p-12">
      <div className="flex flex-col items-center text-center">
        <EmptyHero.Icons className="mb-8">
          <IconEarthOutline18 />
          <IconUserOutline18 />
          <IconWindowLayoutOutline18 />
          <IconKeyOutline18 />
          <IconShieldKeyOutline18 />
        </EmptyHero.Icons>

        <h2 className="text-accent-12 font-semibold text-2xl leading-8 mb-1">Customer portal</h2>
        <p className="text-accent-11 text-sm leading-6 max-w-md text-balance mb-6">
          An Unkey-hosted portal that allows your customers to manage their keys themselves.
        </p>

        <Button variant="primary" size="md" onClick={onEnable}>
          Enable Customer portal
        </Button>
      </div>
    </div>
  );
}
