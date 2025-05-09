
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { FlexibleContainer } from "./flexible-container";
export function ControlsContainer({ children }: { children: React.ReactNode }) {
    return (
        <FlexibleContainer border="bottom" padding="none" borderRadius="none">
            <div className="flex items-center justify-between w-full px-3 py-2 min-h-10">{children}</div>
        </FlexibleContainer>
    );
}

export function ControlsLeft({ children }: { children: React.ReactNode }) {
    return <div className="flex gap-2">{children}</div>;
}

export function ControlsRight({ children }: { children: React.ReactNode }) {
    return <div className="flex gap-2">{children}</div>;
}
