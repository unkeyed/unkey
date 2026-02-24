"use client";
import { BookBookmark, Discord } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const OnboardingLinks = () => (
  <div className="flex gap-3 items-center">
    <Button
      variant="outline"
      className="text-gray-12 text-[13px] font-medium border border-grayA-4 gap-2 rounded-full flex items-center px-3 py-1.5 transition-all"
      onClick={() =>
        window.open("https://www.unkey.com/docs/introduction", "_blank", "noopener,noreferrer")
      }
    >
      <BookBookmark className="text-gray-12 shrink-0 size-[18px]" iconSize="sm-regular" />
      View documentation
    </Button>
    <Button
      variant="outline"
      className="text-gray-12 text-[13px] font-medium border border-grayA-4 gap-2 rounded-full flex items-center px-3 py-1.5 transition-all"
      onClick={() => window.open("https://discord.gg/fDbezjbJbD", "_blank", "noopener,noreferrer")}
    >
      <div className="size-[18px] overflow-hidden flex items-center justify-center">
        <Discord
          className="text-feature-11 shrink-0"
          style={{ width: 18, height: 18 }}
          iconSize="sm-regular"
        />
      </div>
      Join community
    </Button>
  </div>
);
