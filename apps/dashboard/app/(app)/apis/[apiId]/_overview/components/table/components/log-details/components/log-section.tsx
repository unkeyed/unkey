"use client";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Button } from "@unkey/ui";

import { TimestampInfo } from "@/components/timestamp-info";
import { Clone } from "@unkey/icons";
import { isValid, parse, parseISO } from "date-fns";

const TIME_KEYWORDS = [
  "created",
  "created_at",
  "createdAt",
  "updated",
  "updated_at",
  "updatedAt",
  "time",
  "date",
  "timestamp",
  "expires",
  "expired",
  "expiration",
  "last",
  "refill_at",
  "used",
];

export const LogSection = ({
  details,
  title,
}: {
  details: string | string[];
  title: string;
}) => {
  const handleClick = () => {
    navigator.clipboard
      .writeText(getFormattedContent(details))
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">{title}</span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group">
          <pre className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {Array.isArray(details)
              ? details.map((header) => {
                  const [key, ...valueParts] = header.split(":");
                  const value = valueParts.join(":").trim();

                  // Check if this is a timestamp field we should enhance
                  const keyLower = key.toLowerCase();
                  const isTimeField = TIME_KEYWORDS.some((keyword) => keyLower.includes(keyword));
                  const shouldEnhance = isTimeField && isTimeValue(value);

                  return (
                    <div className="group flex items-center w-full p-[3px]" key={key}>
                      <span className="text-left text-accent-9 whitespace-nowrap">
                        {key}
                        {value ? ":" : ""}
                      </span>
                      {shouldEnhance ? (
                        <span className="ml-2 text-xs text-accent-12 truncate">
                          <TimestampInfo value={value} />
                        </span>
                      ) : (
                        <span className="ml-2 text-xs text-accent-12 truncate">{value}</span>
                      )}
                    </div>
                  );
                })
              : details}
          </pre>
          <Button
            shape="square"
            onClick={handleClick}
            variant="outline"
            className="absolute bottom-2 right-3 opacity-0 group-hover:opacity-100 transition-opacity rounded-sm"
            aria-label="Copy content"
          >
            <Clone />
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};

const getFormattedContent = (details: string | string[]) => {
  if (Array.isArray(details)) {
    return details
      .map((header) => {
        const [key, ...valueParts] = header.split(":");
        const value = valueParts.join(":").trim();
        return `${key}: ${value}`;
      })
      .join("\n");
  }
  return details;
};

const isTimeValue = (value: string): boolean => {
  // Skip non-timestamp values
  if (
    value === "N/A" ||
    value === "Invalid Date" ||
    value.startsWith("Less than") ||
    /^\d+ (day|hour|minute|second)s?$/.test(value)
  ) {
    return false;
  }

  try {
    // Handle ISO format strings
    if (/^\d{4}-\d{2}-\d{2}/.test(value)) {
      return isValid(parseISO(value));
    }

    // Handle common localized formats
    if (/\d{1,2}\/\d{1,2}\/\d{4}/.test(value)) {
      // Try US format first: MM/DD/YYYY
      const datePart = value.split(",")[0];
      const parsedDate = parse(datePart, "M/d/yyyy", new Date());
      return isValid(parsedDate);
    }

    // Handle month name formats
    if (/[A-Za-z]{3}\s\d{1,2},\s\d{4}/.test(value)) {
      const parsedDate = parse(value, "MMM d, yyyy", new Date());
      return isValid(parsedDate);
    }

    // Fallback to standard JS date parsing
    const date = new Date(value);
    return !Number.isNaN(date.getTime());
  } catch {
    return false;
  }
};
