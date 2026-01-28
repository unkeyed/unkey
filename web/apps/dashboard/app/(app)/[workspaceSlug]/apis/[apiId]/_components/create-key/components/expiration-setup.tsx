"use client";
import { ProtectionSwitch } from "@/components/dashboard/metadata/protection-switch";
import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { Clock } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { addDays, addMinutes, format } from "date-fns";
import { useState } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { ExpirationFormValues } from "../create-key.schema";

const EXPIRATION_OPTIONS = [
  {
    id: 1,
    display: "1 day",
    value: "1d",
    description: "Key expires in 1 day",
    checked: false,
  },
  {
    id: 2,
    display: "1 week",
    value: "7d",
    description: "Key expires in 1 week",
    checked: false,
  },
  {
    id: 3,
    display: "1 month",
    value: "30d",
    description: "Key expires in 30 days",
    checked: false,
  },
  {
    id: 4,
    display: "Custom",
    value: undefined,
    description: "Set custom expiration date and time",
    checked: false,
  },
];

export const ExpirationSetup = ({
  overrideEnabled = false,
}: {
  overrideEnabled?: boolean;
}) => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
  } = useFormContext<ExpirationFormValues>();

  const [selectedTitle, setSelectedTitle] = useState<string>("1 day");

  const expirationEnabled = useWatch({
    control,
    name: "expiration.enabled",
  });

  const currentExpiryDate = useWatch({
    control,
    name: "expiration.data",
  });

  const handleSwitchChange = (checked: boolean) => {
    setValue("expiration.enabled", checked);

    // Set default expiry date (1 day) when enabling if not already set
    if (checked && !currentExpiryDate) {
      setValue("expiration.data", addDays(new Date(), 1));
    }
  };

  // Calculate minimum valid date (10 minutes from now)
  const minValidDate = addMinutes(new Date(), 10);

  // Handle date and time selection from DatetimePopover
  const handleDateTimeChange = (startTime?: number, _?: number, since?: string) => {
    if (since) {
      // Handle predefined time ranges
      let newDate = new Date();
      switch (since) {
        case "1d":
          newDate = addDays(newDate, 1);
          break;
        case "7d":
          newDate = addDays(newDate, 7);
          break;
        case "30d":
          newDate = addDays(newDate, 30);
          break;
      }
      setValue("expiration.data", newDate);
    } else if (startTime) {
      // Handle custom date selection
      const newDate = new Date(startTime);

      // Check if the date is valid (at least 2 minutes in the future)
      if (newDate < minValidDate) {
        // If date is too soon, set it to minimum valid date
        setValue("expiration.data", minValidDate);
      } else {
        setValue("expiration.data", newDate);
      }
    }
  };

  // Format date for display
  const formatExpiryDate = (date?: Date) => {
    if (!date) {
      return "Select expiration date";
    }
    return format(date, "MMM d, yyyy 'at' h:mm a");
  };

  const getInitialTimeValues = () => {
    // Safely convert currentExpiryDate to Date object, fallback to minValidDate
    // Type guard: only use currentExpiryDate if it's actually a Date
    const initialDate =
      currentExpiryDate instanceof Date ? new Date(currentExpiryDate) : minValidDate;

    // If conversion failed, use minValidDate
    const safeInitialDate = Number.isNaN(initialDate.getTime()) ? minValidDate : initialDate;

    return {
      startTime: safeInitialDate.getTime(),
      endTime: undefined,
      since: undefined,
    };
  };

  // Calculate date for showing warning about close expiry (less than 1 hour)
  const isExpiringVerySoon =
    currentExpiryDate instanceof Date &&
    new Date(currentExpiryDate).getTime() - Date.now() < 60 * 60 * 1000;

  const getExpiryDescription = () => {
    if (isExpiringVerySoon) {
      return "This key will expire very soon (less than 1 hour). Consider setting a longer expiration time.";
    }
    return "The key will be automatically disabled at the specified date and time (UTC).";
  };

  return (
    <div className="space-y-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description="Turn on to set an expiration date. When reached, the key will be automatically disabled."
          title="Expiration"
          icon={<Clock className="text-gray-12" iconSize="sm-regular" />}
          checked={expirationEnabled}
          onCheckedChange={handleSwitchChange}
          {...register("expiration.enabled")}
        />
      )}

      <Controller
        control={control}
        name="expiration.data"
        render={({ field }) => (
          <DatetimePopover
            initialTitle={selectedTitle}
            initialTimeValues={getInitialTimeValues()}
            onSuggestionChange={setSelectedTitle}
            onDateTimeChange={handleDateTimeChange}
            customOptions={EXPIRATION_OPTIONS}
            customHeader={<ExpirationHeader />}
            singleDateMode
            minDate={minValidDate} // Set minimum date to 2 minutes from now
          >
            <FormInput
              label="Expiry Date"
              description={getExpiryDescription()}
              readOnly
              disabled={!expirationEnabled}
              value={formatExpiryDate(field.value as Date | undefined)}
              className="cursor-pointer w-full"
              variant={expirationEnabled && isExpiringVerySoon ? "warning" : undefined}
              error={
                errors.expiration?.data && "message" in errors.expiration.data
                  ? errors.expiration.data.message
                  : undefined
              }
            />
          </DatetimePopover>
        )}
      />
    </div>
  );
};

const ExpirationHeader = () => {
  return (
    <div className="flex justify-between w-full h-8 px-2">
      <span className="text-gray-9 text-[13px] w-full">Choose expiration date</span>
    </div>
  );
};
