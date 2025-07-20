'use client';

import { signIn } from "next-auth/react";

interface SignInButtonProps {
  provider: string;
  children: React.ReactNode;
  callbackUrl?: string;
}

export function SignInButton({ provider, children, callbackUrl = "/dashboard" }: SignInButtonProps) {
  const handleSignIn = async () => {
    try {
      await signIn(provider, { callbackUrl });
    } catch (error) {
      console.error("Sign in error:", error);
    }
  };

  return (
    <div onClick={handleSignIn} className="cursor-pointer">
      {children}
    </div>
  );
}