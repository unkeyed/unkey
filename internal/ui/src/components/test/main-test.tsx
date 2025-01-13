"use client";

import type React from "react";
import { createContext, useContext, useState } from "react";

interface NameContextType {
  name: string | undefined;
  changeName: (name: string) => void;
}
const Namecontext = createContext<NameContextType>({
  name: "Mike",
  changeName: () => {}, // Empty function as a default
});

type MainProps = React.FC & {
  children: React.ReactNode;
};

function MainTest({ children }: MainProps) {
  const [name, setName] = useState<string>("Mike");

  const changeName = (name: string) => {
    setName(name);
  };

  return <Namecontext.Provider value={{ name, changeName }}>{children}</Namecontext.Provider>;
}

MainTest.displayName = "MainTest";

const ChildItem: React.FC = () => {
  const { name } = useContext(Namecontext);
  return <div>{name}</div>;
};
MainTest.ChildItem = ChildItem;
const ChildInput: React.FC = () => {
  const { name, changeName } = useContext(Namecontext);

  return (
    <div>
      <form>
        <input type="text" value={name} onChange={(e) => changeName(e.target.value)} />
      </form>
    </div>
  );
};

export { MainTest, ChildItem, ChildInput };
