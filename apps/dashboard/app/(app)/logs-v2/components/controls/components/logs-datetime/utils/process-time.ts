type TimeUnit = {
  HH?: string;
  mm?: string;
  ss?: string;
};

//Process new Date and time filters to be added to the filters as time since epoch
export const processTimeFilters = (date?: Date, newTime?: TimeUnit) => {
  if (date) {
    const hours = newTime?.HH ? Number.parseInt(newTime.HH) : 0;
    const minutes = newTime?.mm ? Number.parseInt(newTime.mm) : 0;
    const seconds = newTime?.ss ? Number.parseInt(newTime.ss) : 0;
    date.setHours(hours, minutes, seconds, 0);
    return date;
  }
};
