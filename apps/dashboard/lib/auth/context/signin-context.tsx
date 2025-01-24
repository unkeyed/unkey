import { createContext, useContext, useState, ReactNode } from "react";

interface SignInContext {
  error: string | null;
  setError: (error: string | null) => void;
  isVerifying: boolean;
  setIsVerifying: (verifying: boolean) => void;
  email: string;
  setEmail: (email: string) => void;
  accountNotFound: boolean;
  setAccountNotFound: (notFound: boolean) => void;
}

export const SignInContext = createContext<SignInContext | undefined>(undefined);

export function SignInProvider({ children }: { children: ReactNode }) {
  const [error, setError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [email, setEmail] = useState("");
  const [accountNotFound, setAccountNotFound] = useState(false);

  return (
    <SignInContext.Provider
      value={{
        error,
        setError,
        isVerifying,
        setIsVerifying,
        email,
        setEmail,
        accountNotFound,
        setAccountNotFound,
      }}
    >
      {children}
    </SignInContext.Provider>
  );
}
