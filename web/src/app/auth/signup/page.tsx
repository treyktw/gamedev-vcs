'use client'

import { Suspense } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { GitBranch, Github, Chrome, Sparkles, Zap, Lock, Users, BarChart3 } from "lucide-react";
import Link from "next/link";
import Image from "next/image";
import { SignInButton } from "@/components/auth/signin-button";


function SignUpForm() {
  return (
    <div className="min-h-screen flex">
      {/* Left side - Background Image */}
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

      {/* Right side - Sign Up Form */}
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
            
            <div className="flex items-center justify-center space-x-2 mb-4">
              <Badge variant="secondary" className="text-sm flex items-center gap-2">
              <Sparkles className="w-10 h-10 text-primary" />
                Free for indie teams
              </Badge>
            </div>
            
            <h1 className="text-3xl font-bold tracking-tight">
              Create Your Account
            </h1>
            <p className="text-muted-foreground">
              Start collaborating on your game projects today
            </p>
          </div>

          {/* Sign Up Options */}
          <Card className="border-2 shadow-lg">
            <CardHeader className="text-center pb-4">
              <CardTitle className="text-lg">Get started with</CardTitle>
              <CardDescription>
                Choose your preferred sign-up method
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Google Sign Up */}
              <SignInButton provider="google">
                <Button
                  variant="outline"
                  size="lg"
                  className="w-full border-2 hover:bg-accent hover:shadow-md transition-all"
                >
                  <Chrome className="w-5 h-5 mr-3 text-[#4285f4]" />
                  Sign up with Google
                </Button>
              </SignInButton>

              {/* GitHub Sign Up */}
              <SignInButton provider="github">
                <Button
                  variant="outline"
                  size="lg"
                  className="w-full border-2 hover:bg-accent hover:shadow-md transition-all"
                >
                  <Github className="w-5 h-5 mr-3" />
                  Sign up with GitHub
                </Button>
              </SignInButton>

              <div className="relative">
                <Separator className="my-4" />
                <div className="absolute inset-0 flex items-center justify-center">
                  <Badge variant="secondary" className="px-3 bg-background text-primary">
                    or
                  </Badge>
                </div>
              </div>

              {/* Already have account */}
              <div className="p-4 bg-muted/50 rounded-lg border">
                <p className="text-sm font-medium mb-2">🔑 Already have an account?</p>
                <p className="text-xs text-muted-foreground mb-3">
                  Sign in to access your projects and continue where you left off
                </p>
                <Link href="/auth/signin">
                  <Button variant="secondary" size="sm" className="w-full">
                    Sign In Instead
                  </Button>
                </Link>
              </div>
            </CardContent>
          </Card>

          {/* Footer */}
          <div className="text-center text-sm text-muted-foreground">
            <p>
              By creating an account, you agree to our{" "}
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
    </div>
  );
}

export default function SignUpPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <SignUpForm />
    </Suspense>
  );
}