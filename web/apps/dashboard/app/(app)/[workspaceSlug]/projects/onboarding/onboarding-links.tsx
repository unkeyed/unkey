import { BookBookmark, Discord } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const OnboardingLinks = () => (
  <div className="flex gap-3 items-center">
    <Button
      variant="outline"
      className="text-gray-12 text-[13px] font-medium border border-grayA-4 rounded-full px-3 py-1.5 transition-all "
    >
      <a
        href="https://www.unkey.com/docs/introduction"
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center  w-full gap-2"
      >
        <BookBookmark className="text-gray-12 shrink-0 size-[18px]" iconSize="sm-regular" />
        View documentation
      </a>
    </Button>
    <Button
      variant="outline"
      className="text-gray-12 text-[13px] font-medium border border-grayA-4 rounded-full px-3 py-1.5 transition-all "
    >
      <a
        href="https://unkey.com/discord"
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center  w-full gap-2"
      >
        <div className="size-[18px] overflow-hidden flex items-center justify-center">
          <Discord
            className="text-feature-11 shrink-0"
            style={{ width: 18, height: 18 }}
            iconSize="sm-regular"
          />
        </div>
        Join community
      </a>
    </Button>
  </div>
);
