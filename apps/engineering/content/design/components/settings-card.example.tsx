"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Clone } from "@unkey/icons";
import { Button, Input, SettingCard } from "@unkey/ui";

export const SettingsCardBasic = () => {
    return (
        <RenderComponentWithSnippet>
            <SettingCard
                title="Example Name"
                description="The name you give your API."
                border="both"
            >
                <div className="flex gap-2 items-center justify-center w-full">
                    <Input
                        placeholder="API name"
                        value="My-API"
                    />
                    <Button
                        size="lg"
                    >
                        Save
                    </Button>
                </div>
            </SettingCard>
        </RenderComponentWithSnippet>
    );
};

export const SettingsCardsWithSharedEdge = () => {
    return (
        <RenderComponentWithSnippet>
            <div>
                <SettingCard
                    title="Example length"
                    description="Number value input with a save button."
                    border="top"
                    className="border-b"
                >
                    <div className="flex gap-2 items-center justify-center w-full">
                        <Input
                            placeholder="size"
                            value="16"
                            type="number"
                            className="w-full"
                        />
                        <Button
                            size="lg"
                        >
                            Save
                        </Button>
                    </div>
                </SettingCard>
                <SettingCard
                    title="Example ID"
                    description="ID that can be copied to clipboard."
                    border="bottom"
                >
                    <Input
                        readOnly
                        disabled
                        defaultValue={"Key_1234567890"}
                        rightIcon={
                            <button type="button">
                                <Clone size="sm-regular" />
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
        <RenderComponentWithSnippet>
            <div>
                <SettingCard
                    title="Example length"
                    description="Number value input with a save button."
                >
                    <div className="flex gap-2 items-center justify-center w-full">
                        <Input
                            placeholder="size"
                            value="16"
                            type="number"
                            className="w-full"
                        />
                        <Button
                            size="lg"
                        >
                            Save
                        </Button>
                    </div>
                </SettingCard>
            </div>
        </RenderComponentWithSnippet>
    );
};

