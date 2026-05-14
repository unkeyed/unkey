import { Empty } from "../../../empty";

export function EmptyApiRequests() {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-100 flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>No verifications in this time range</Empty.Title>
        <Empty.Description className="text-left">
          This API has verification history, but none in the selected time range. Try a wider range
          or check your active filters.
        </Empty.Description>
      </Empty>
    </div>
  );
}
