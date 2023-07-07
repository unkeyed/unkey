import { PropsWithChildren } from "react";

const MessageContainer = ({ children }: PropsWithChildren) => {
  return (
    <div className="h-[70vh] p-12 shadow-fuller-shadow w-5/6 overflow-y-scroll">
      {children}
    </div>
  );
};

export default MessageContainer;
