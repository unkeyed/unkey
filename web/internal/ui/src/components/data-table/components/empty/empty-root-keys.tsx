import { BookBookmark } from "@unkey/icons";
import { buttonVariants } from "../../../buttons/button";
import { Empty } from "../../../empty";

export function EmptyRootKeys() {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-[400px] flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>No root keys found</Empty.Title>
        <Empty.Description className="text-left">
          There are no root keys configured yet. Create your first root key to start managing
          permissions and access control.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-start">
          <a
            href="https://www.unkey.com/docs/security/overview#root-keys"
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ variant: "outline" })}
          >
            <span className="flex items-center gap-2">
              <BookBookmark />
              Learn about root keys
            </span>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
}
