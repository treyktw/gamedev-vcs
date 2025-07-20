import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { GitBranch, Users, Lock, Activity, BarChart3, Zap } from "lucide-react";
import Link from "next/link";

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-primary rounded-md flex items-center justify-center">
                <GitBranch className="w-5 h-5 text-primary-foreground" />
              </div>
              <span className="text-xl font-bold">GameDev VCS</span>
            </div>
            <div className="flex items-center space-x-4">
              <Link href="/auth/signin">
                <Button variant="ghost">Sign In</Button>
              </Link>
              <Link href="/auth/signup">
                <Button>Get Started</Button>
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="container mx-auto px-4 py-16 text-center">
        <div className="max-w-4xl mx-auto">
          <Badge variant="secondary" className="mb-4">
            Built for Game Development Teams
          </Badge>
          <h1 className="text-4xl md:text-6xl font-bold mb-6 bg-gradient-to-r from-foreground to-muted-foreground bg-clip-text text-transparent">
            Version Control
            <br />
            That Actually Works
            <br />
            For Game Assets
          </h1>
          <p className="text-xl text-muted-foreground mb-8 max-w-2xl mx-auto">
            Real-time collaboration, binary asset handling, and UE5 integration. 
            Everything Git can't do for game development teams.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/auth/signup">
              <Button size="lg" className="w-full sm:w-auto">
                Start Your Project
              </Button>
            </Link>
            <Link href="/demo">
              <Button variant="outline" size="lg" className="w-full sm:w-auto">
                View Demo
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Features Grid */}
      <section className="container mx-auto px-4 py-16">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold mb-4">Built for Game Development</h2>
          <p className="text-muted-foreground max-w-2xl mx-auto">
            Every feature designed specifically for the challenges of game asset management
          </p>
        </div>
        
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          <Card>
            <CardHeader>
              <Lock className="w-8 h-8 text-primary mb-2" />
              <CardTitle>Real-time File Locking</CardTitle>
              <CardDescription>
                Prevent merge conflicts on binary assets with exclusive file locking
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <Users className="w-8 h-8 text-primary mb-2" />
              <CardTitle>Live Collaboration</CardTitle>
              <CardDescription>
                See who's working on what in real-time across your entire project
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <Zap className="w-8 h-8 text-primary mb-2" />
              <CardTitle>UE5 Integration</CardTitle>
              <CardDescription>
                Native support for .uasset files with dependency tracking
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <Activity className="w-8 h-8 text-primary mb-2" />
              <CardTitle>Asset Dependencies</CardTitle>
              <CardDescription>
                Automatic dependency graph tracking prevents broken references
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <BarChart3 className="w-8 h-8 text-primary mb-2" />
              <CardTitle>Team Analytics</CardTitle>
              <CardDescription>
                Productivity insights and asset hotspot analysis for your team
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <GitBranch className="w-8 h-8 text-primary mb-2" />
              <CardTitle>Smart Branching</CardTitle>
              <CardDescription>
                Branch management designed for large binary assets and game projects
              </CardDescription>
            </CardHeader>
          </Card>
        </div>
      </section>

      {/* Current Team Activity Preview */}
      <section className="container mx-auto px-4 py-16">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Live Team Activity</h2>
          
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center space-x-2">
                <Activity className="w-5 h-5" />
                <span>awesome-game</span>
                <Badge variant="secondary">7 members online</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Mock team activity */}
              <div className="flex items-center space-x-3 p-3 rounded-lg bg-muted/50">
                <Avatar className="w-8 h-8">
                  <AvatarFallback>AL</AvatarFallback>
                </Avatar>
                <div className="flex-1">
                  <p className="text-sm font-medium">Alice is editing</p>
                  <p className="text-xs text-muted-foreground">Assets/Characters/Hero.uasset</p>
                </div>
                <Badge variant="destructive" className="text-xs">Locked</Badge>
              </div>

              <div className="flex items-center space-x-3 p-3 rounded-lg bg-muted/50">
                <Avatar className="w-8 h-8">
                  <AvatarFallback>BT</AvatarFallback>
                </Avatar>
                <div className="flex-1">
                  <p className="text-sm font-medium">Bob committed changes</p>
                  <p className="text-xs text-muted-foreground">Updated level lighting setup</p>
                </div>
                <Badge variant="secondary" className="text-xs">2 min ago</Badge>
              </div>

              <div className="flex items-center space-x-3 p-3 rounded-lg bg-muted/50">
                <Avatar className="w-8 h-8">
                  <AvatarFallback>CH</AvatarFallback>
                </Avatar>
                <div className="flex-1">
                  <p className="text-sm font-medium">Charlie is viewing</p>
                  <p className="text-xs text-muted-foreground">Assets/Materials/Shaders/</p>
                </div>
                <Badge variant="outline" className="text-xs">Online</Badge>
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      <Separator className="my-16" />

      {/* Footer */}
      <footer className="container mx-auto px-4 py-8">
        <div className="flex flex-col md:flex-row justify-between items-center">
          <div className="flex items-center space-x-2 mb-4 md:mb-0">
            <div className="w-6 h-6 bg-primary rounded-md flex items-center justify-center">
              <GitBranch className="w-4 h-4 text-primary-foreground" />
            </div>
            <span className="font-semibold">GameDev VCS</span>
          </div>
          <p className="text-sm text-muted-foreground">
            Built for game development teams. Open source.
          </p>
        </div>
      </footer>
    </div>
  );
}