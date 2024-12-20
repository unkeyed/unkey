import { eventGroups } from "./constants";
export const getEventType = (event: string) => {
  //@ts-expect-error passing the event as string to make it easier to use
  if (eventGroups.create.includes(event)) {
    return "create";
  }
  //@ts-expect-error passing the event as string to make it easier to use
  if (eventGroups.delete.includes(event)) {
    return "delete";
  }
  //@ts-expect-error passing the event as string to make it easier to use
  if (eventGroups.update.includes(event)) {
    return "update";
  }
  return "other";
};
