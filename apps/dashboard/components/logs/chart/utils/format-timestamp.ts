import { addMinutes, format } from "date-fns";

export const formatTimestampTooltip = (value: string | number) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);
  return format(localDate, "MMM dd HH:mm aa");
};

export const formatTimestampLabel = (timestamp: string | number | Date) => {
  const date = new Date(timestamp);
  return format(date, "MMM dd, h:mma").toUpperCase();
};
