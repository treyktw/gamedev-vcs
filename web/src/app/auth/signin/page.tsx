"use client";

import { Suspense } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { GitBranch, Github, Chrome, Zap, Lock, Users } from "lucide-react";
import Link from "next/link";
import Image from "next/image";
import { SignInButton } from "@/components/auth/signin-button";

function SignInForm() {
  return (
    <div className="min-h-screen flex">
      {/* Left side - Login Form */}
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-md space-y-8">
          {/* Header */}
          <div className="text-center space-y-2">
            <div className="flex items-center justify-center space-x-2 mb-6">
              <div className="w-10 h-10 bg-primary rounded-md flex items-center justify-center shadow-lg">
                <GitBranch className="w-6 h-6 text-primary-foreground" />
              </div>
              <span className="text-2xl font-bold">GameDev VCS</span>
            </div>

            <h1 className="text-3xl font-bold tracking-tight">Welcome Back</h1>
            <p className="text-muted-foreground">
              Sign in to your account to continue building
            </p>
          </div>

          {/* Auth Providers */}
          <Card className="border-2 shadow-lg">
            <CardHeader className="text-center pb-4">
              <CardTitle className="text-lg">Sign in with</CardTitle>
              <CardDescription>
                Choose your preferred authentication method
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Google Sign In */}
              <SignInButton provider="google">
                <Button
                  variant="outline"
                  size="lg"
                  className="w-full border-2 hover:bg-accent hover:shadow-md transition-all"
                >
                  <Chrome className="w-5 h-5 mr-3 text-[#4285f4]" />
                  Continue with Google
                </Button>
              </SignInButton>

              {/* GitHub Sign In */}
              <SignInButton provider="github">
                <Button
                  variant="outline"
                  size="lg"
                  className="w-full border-2 hover:bg-accent hover:shadow-md transition-all"
                >
                  <Github className="w-5 h-5 mr-3" />
                  Continue with GitHub
                </Button>
              </SignInButton>

              <div className="relative">
                <Separator className="my-4" />
                <div className="absolute inset-0 flex items-center justify-center">
                  <Badge variant="secondary" className="px-3 bg-background">
                    or
                  </Badge>
                </div>
              </div>

              {/* Demo Account */}
              <div className="p-4 bg-muted/50 rounded-lg border">
                <p className="text-sm font-medium mb-2">
                  👋 New to GameDev VCS?
                </p>
                <p className="text-xs text-muted-foreground mb-3">
                  Sign up with GitHub or Google to get started with your first
                  project
                </p>
                <Link href="/auth/signup">
                  <Button variant="secondary" size="sm" className="w-full">
                    Create Account
                  </Button>
                </Link>
              </div>
            </CardContent>
          </Card>

          {/* Footer */}
          <div className="text-center text-sm text-muted-foreground">
            <p>
              By signing in, you agree to our{" "}
              <Link href="/terms" className="underline hover:text-foreground">
                Terms of Service
              </Link>{" "}
              and{" "}
              <Link href="/privacy" className="underline hover:text-foreground">
                Privacy Policy
              </Link>
            </p>
          </div>
        </div>
      </div>

      {/* Right side - Background Image */}
      <div className="hidden lg:flex flex-1 relative">
        <div className="absolute inset-0 bg-gradient-to-br from-primary/20 to-accent/20" />
        {/* Uncomment and use this when you have your background image */}

        <Image
          src="/background.png"
          alt="Game development workflow"
          fill
          className="object-cover"
          priority
        />
        <div className="absolute inset-0 bg-gradient-to-br from-background/80 to-background/40" />

        <div className="relative z-10 flex items-center justify-center p-12 w-full h-full">
          <div className="text-center space-y-6">
            <h2 className="text-4xl font-bold text-white flex items-center justify-center gap-2 flex-col">
              <GitBranch className="w-16 h-16 text-primary bg-primary/20 rounded-full p-2" />
              Built for Game Development
            </h2>
            <p className="text-xl text-white/90 max-w-lg">
              Version control that actually works with binary assets, real-time
              collaboration, and UE5 integration.
            </p>
            <div className="flex justify-center space-x-4">
              <Badge variant="secondary" className="text-sm">
                🔒 File Locking
              </Badge>
              <Badge variant="secondary" className="text-sm">
                ⚡ Real-time
              </Badge>
              <Badge variant="secondary" className="text-sm">
                🎮 UE5 Ready
              </Badge>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function SignInPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <SignInForm />
    </Suspense>
  );
}
