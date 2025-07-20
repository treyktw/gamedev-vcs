"use client";

import { signOut } from "next-auth/react";
import {
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import { LogOut } from "lucide-react";

export function SignOutButton() {
  return (
    <DropdownMenuItem 
      className="text-destructive focus:text-destructive" 
      onClick={() => signOut()}
    >
      <LogOut className="w-4 h-4 mr-2" />
      Sign out
    </DropdownMenuItem>
  );
} 