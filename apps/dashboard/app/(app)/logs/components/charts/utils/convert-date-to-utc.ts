import { addMinutes } from "date-fns";

export const convertSelectedDateToUTC = (value: Date | number) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);
  return localDate.getTime();
};
