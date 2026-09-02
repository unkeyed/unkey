import { Button, Empty } from "@unkey/ui";
import { IconBookBookmarkOutline18 } from "nucleo-ui-outline-18";

export const EmptyPermissions = () => {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <Empty className="w-[400px] flex items-start">
        <Empty.Icon className="w-auto" />
        <Empty.Title>No Permissions Found</Empty.Title>
        <Empty.Description className="text-left">
          There are no permissions configured yet. Create your first permission to start managing
          permissions and access control.
        </Empty.Description>
        <Empty.Actions className="mt-4 justify-start">
          <a
            href="https://www.unkey.com/docs/platform/apis/features/authorization/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button size="md">
              <IconBookBookmarkOutline18 />
              Learn about Permissions
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  );
};
