import { useState } from 'react';
import { resendAuthCode, verifyAuthCode } from '../actions';
import { UserData } from '../types';

interface VerificationStatus {
  isVerifying: boolean;
  isVerified: boolean;
  error: string | null;
}

export function useSignUp() {
    const [userData, setUserData] = useState<UserData>({
      firstName: '',
      lastName: '',
      email: ''
    });
  
    const [verificationStatus, setVerificationStatus] = useState<VerificationStatus>({
      isVerifying: false,
      isVerified: false,
      error: null
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
      setVerificationStatus({
        isVerifying: false,
        isVerified: false,
        error: null
      });
    };
  
    const handleVerification = async (code: string): Promise<void> => {
      try {
        setVerificationStatus(prev => ({
          ...prev,
          isVerifying: true,
          error: null
        }));
  
        await verifyAuthCode({
          email: userData.email,
          code
        });
        
        setVerificationStatus({
          isVerifying: false,
          isVerified: true,
          error: null
        });
      } catch (error) {
        setVerificationStatus({
          isVerifying: false,
          isVerified: false,
          error: error instanceof Error ? error.message : 'Verification failed'
        });
        throw error;
      }
    };
  
    const handleResendCode = async (): Promise<void> => {
      try {
        await resendAuthCode(userData.email);
        
        setVerificationStatus({
          isVerifying: false,
          isVerified: false,
          error: null
        });
      } catch (error) {
        setVerificationStatus(prev => ({
          ...prev,
          error: error instanceof Error ? error.message : 'Failed to resend code'
        }));
        throw error;
      }
    };
  
    return {
      userData,
      verificationStatus,
      updateUserData,
      clearUserData,
      handleVerification,
      handleResendCode
    };
  }