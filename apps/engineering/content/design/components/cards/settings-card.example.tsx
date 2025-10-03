"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Clone } from "@unkey/icons";
import { Button, Input, SettingCard } from "@unkey/ui";

export const SettingsCardBasic = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<SettingCard
  title="Example Card Border Both"
  description="Passing border='both' will show a top and bottom border. Content for right side is passed in as a child."
  border="both"
  contentWidth="w-full lg:w-1/2"
>
  <div className="flex gap-2 items-center justify-center w-full">
    <Input placeholder="API name" value="My-API" className="w-full" />
    <Button size="lg">Save</Button>
  </div>
</SettingCard>`}
    >
      <SettingCard
        title="Example Card Border Both"
        description="Passing border='both' will show a top and bottom border. Content for right side is passed in as a child."
        border="both"
        contentWidth="w-full lg:w-1/2"
      >
        <div className="flex gap-2 items-center justify-center w-full">
          <Input placeholder="API name" value="My-API" className="w-full" />
          <Button size="lg">Save</Button>
        </div>
      </SettingCard>
    </RenderComponentWithSnippet>
  );
};

export const SettingsCardsWithSharedEdge = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div>
  <SettingCard
    title="Example Top Card Border"
    description="Passing border='top' will only show a top border."
    border="top"
    contentWidth="w-full lg:w-1/2"
  >
    <div className="flex gap-2 items-center justify-center w-full">
      <Input placeholder="size" value="16" type="number" className="w-full" />
      <Button size="lg">Save</Button>
    </div>
  </SettingCard>
  <SettingCard
    title="Example Card Bottom Border"
    description="Passing border='bottom' will only show a bottom border."
    border="bottom"
    contentWidth="w-full lg:w-1/2"
  >
    <Input
      readOnly
      disabled
      defaultValue={"Key_1234567890"}
      rightIcon={
        <button type="button">
          <Clone iconsize="sm-regular" />
        </button>
      }
    />
  </SettingCard>
</div>`}
    >
      <div>
        <SettingCard
          title="Example Top Card Border"
          description="Passing border='top' will only show a top border."
          border="top"
          contentWidth="w-full lg:w-1/2"
        >
          <div className="flex gap-2 items-center justify-center w-full">
            <Input placeholder="size" value="16" type="number" className="w-full" />
            <Button size="lg">Save</Button>
          </div>
        </SettingCard>
        <SettingCard
          title="Example Card Bottom Border"
          description="Passing border='bottom' will only show a bottom border."
          border="bottom"
          contentWidth="w-full lg:w-1/2"
        >
          <Input
            readOnly
            disabled
            defaultValue={"Key_1234567890"}
            rightIcon={
              <button type="button">
                <Clone iconsize="sm-regular" />
              </button>
            }
          />
        </SettingCard>
      </div>
    </RenderComponentWithSnippet>
  );
};

export const SettingsCardsWithDivider = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div>
  <SettingCard
    title="Example Top Card Border"
    description="Passing border='top' will only show a top border."
    border="top"
    contentWidth="w-full lg:w-1/2"
    className="border-b"
  >
    <div className="flex gap-2 items-center justify-center w-full">
      <Input placeholder="size" value="16" type="number" className="w-full" />
      <Button size="lg">Save</Button>
    </div>
  </SettingCard>
  <SettingCard
    title="Example Card Bottom Border"
    description="Passing border='bottom' will only show a bottom border."
    border="bottom"
    contentWidth="w-full lg:w-1/2"
  >
    <Input
      readOnly
      disabled
      defaultValue={"Key_1234567890"}
      rightIcon={
        <button type="button">
          <Clone iconsize="sm-regular" />
        </button>
      }
    />
  </SettingCard>
</div>`}
    >
      <div>
        <SettingCard
          title="Example Top Card Border"
          description="Passing border='top' will only show a top border."
          border="top"
          contentWidth="w-full lg:w-1/2"
          className="border-b"
        >
          <div className="flex gap-2 items-center justify-center w-full">
            <Input placeholder="size" value="16" type="number" className="w-full" />
            <Button size="lg">Save</Button>
          </div>
        </SettingCard>
        <SettingCard
          title="Example Card Bottom Border"
          description="Passing border='bottom' will only show a bottom border."
          border="bottom"
          contentWidth="w-full lg:w-1/2"
        >
          <Input
            readOnly
            disabled
            defaultValue={"Key_1234567890"}
            rightIcon={
              <button type="button">
                <Clone iconsize="sm-regular" />
              </button>
            }
          />
        </SettingCard>
      </div>
    </RenderComponentWithSnippet>
  );
};

export const SettingsCardsWithSquareEdge = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div>
  <SettingCard
    title="Square corncers Example"
    description="Not passing in border prop will default to rounded corners."
    contentWidth="w-full lg:w-1/2"
  >
    <div className="flex gap-2 items-center justify-center w-full">
      <Input placeholder="size" value="44" type="number" className="w-full" />
      <Button size="lg">Save</Button>
    </div>
  </SettingCard>
</div>`}
    >
      <div>
        <SettingCard
          title="Square corncers Example"
          description="Not passing in border prop will default to rounded corners."
          contentWidth="w-full lg:w-1/2"
        >
          <div className="flex gap-2 items-center justify-center w-full">
            <Input placeholder="size" value="44" type="number" className="w-full" />
            <Button size="lg">Save</Button>
          </div>
        </SettingCard>
      </div>
    </RenderComponentWithSnippet>
  );
};
