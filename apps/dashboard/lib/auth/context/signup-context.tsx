"use client";

import { createContext, useContext, useState } from 'react';
import { UserData } from '../types';

interface SignUpContextType {
  userData: UserData;
  updateUserData: (data: Partial<UserData>) => void;
  clearUserData: () => void;
}

export const SignUpContext = createContext<SignUpContextType>({} as SignUpContextType);

export function SignUpProvider({ children }: { children: React.ReactNode }) {
  const [userData, setUserData] = useState<UserData>({
    firstName: '',
    lastName: '',
    email: ''
  });

  const updateUserData = (newData: Partial<UserData>) => {
    setUserData(prev => ({
      ...prev,
      ...newData
    }));
  };

  const clearUserData = () => {
    setUserData({
      firstName: '',
      lastName: '',
      email: ''
    });
  };

  return (
    <SignUpContext.Provider value={{ userData, updateUserData, clearUserData }}>
      {children}
    </SignUpContext.Provider>
  );
}

export function useSignUpContext() {
  const context = useContext(SignUpContext);
  if (!context) {
    throw new Error('useSignUpContext must be used within a SignUpProvider');
  }
  return context;
}