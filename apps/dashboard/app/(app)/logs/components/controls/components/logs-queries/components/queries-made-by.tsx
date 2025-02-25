import { Avatar, AvatarImage } from "@/components/ui/avatar";
import { CircleHalfDottedClock } from "@unkey/icons";

type QueriesMadeByProps = {
  userName?: string;
  userImageSrc: string;
  createdString: string;
};

export const QueriesMadeBy = ({ userName, userImageSrc, createdString }: QueriesMadeByProps) => {
  return (
    <div className="flex flex-row w-full justify-start items-center h-6 gap-2 mt-2">
      {/* User Avatar */}
      {userName && <span className="font-mono font-normal text-xs text-gray-9">by</span>}
      {userName && (
        <Avatar className="h-[21px] w-[21px]">
          <AvatarImage
            src={userImageSrc}
            alt={userName}
            className="rounded-full border border-gray-4 border-[1px] "
          />
        </Avatar>
      )}
      {userName && (
        <span className="font-mono font-medium leading-4 text-xs text-gray-12">{userName}</span>
      )}
      <CircleHalfDottedClock className="size-3.5 text-gray-12 mb-[2px] ml-[2px]" />
      <span className="font-mono font-normal text-xs leading-4 text-gray-9">{createdString}</span>
    </div>
  );
};
