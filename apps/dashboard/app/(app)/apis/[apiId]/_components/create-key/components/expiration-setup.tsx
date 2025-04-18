"use client";
import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { Switch } from "@/components/ui/switch";
import { Clock } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { addDays, addMinutes, format } from "date-fns";
import { useState } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { ExpirationFormValues } from "../schema";

const ExpirationHeader = () => {
  return (
    <div className="flex justify-between w-full h-8 px-2">
      <span className="text-gray-9 text-[13px] w-full">Choose expiration date</span>
    </div>
  );
};

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

export const ExpirationSetup = () => {
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
    defaultValue: false,
  });

  const currentExpiryDate = useWatch({
    control,
    name: "expiration.data",
  });

  const handleSwitchChange = (checked: boolean) => {
    setValue("expiration.enabled", checked);
  };

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
      const minValidDate = addMinutes(new Date(), 2);

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

  // Calculate initial time values for DatetimePopover
  const getInitialTimeValues = () => {
    return {
      startTime: currentExpiryDate?.getTime(),
      endTime: undefined,
      since: undefined,
    };
  };

  // Calculate date for showing warning about close expiry (less than 1 hour)
  const isExpiringVerySoon =
    currentExpiryDate && currentExpiryDate.getTime() - new Date().getTime() < 60 * 60 * 1000;

  const getExpiryDescription = () => {
    if (isExpiringVerySoon) {
      return "This key will expire very soon (less than 1 hour). Consider setting a longer expiration time.";
    }
    return "The key will be automatically disabled at the specified date and time (UTC).";
  };

  return (
    <div className="space-y-5 px-2 py-1">
      <div className="flex flex-row py-5 pl-5 pr-[26px] gap-14 justify-between border rounded-xl border-grayA-5 bg-white dark:bg-black items-center">
        <div className="flex flex-col gap-4">
          <div className="flex gap-3">
            <div className="p-1.5 bg-grayA-3 rounded-md border border-grayA-3">
              <Clock className="text-gray-12" size="sm-regular" />
            </div>
            <div className="text-sm font-medium text-gray-12">Expiration</div>
          </div>
          <div className="text-gray-9 text-xs">
            Turn on to set an expiration date. When reached, the key will be automatically disabled.
          </div>
        </div>
        <Switch
          checked={expirationEnabled}
          onCheckedChange={handleSwitchChange}
          className="
            h-4 w-7
            data-[state=checked]:bg-success-9
            data-[state=checked]:ring-2
            data-[state=checked]:ring-successA-5
            data-[state=unchecked]:bg-gray-3
            data-[state=unchecked]:ring-2
            data-[state=unchecked]:ring-grayA-3
            [&>span]:h-3.5 [&>span]:w-3.5
          "
          {...register("expiration.enabled")}
        />
      </div>

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
          >
            <FormInput
              label="Expiry Date"
              description={getExpiryDescription()}
              readOnly
              disabled={!expirationEnabled}
              value={formatExpiryDate(field.value)}
              className="cursor-pointer"
              variant={isExpiringVerySoon ? "warning" : undefined}
              error={errors.expiration?.data?.message}
            />
          </DatetimePopover>
        )}
      />
    </div>
  );
};
