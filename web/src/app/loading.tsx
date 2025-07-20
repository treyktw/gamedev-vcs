import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default function Loading() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header Skeleton */}
      <div className="border-b border-border">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Skeleton className="w-8 h-8 rounded-md" />
              <Skeleton className="w-32 h-6" />
            </div>
            <div className="flex items-center space-x-4">
              <Skeleton className="w-16 h-9" />
              <Skeleton className="w-24 h-9" />
            </div>
          </div>
        </div>
      </div>

      {/* Main Content Skeleton */}
      <div className="container mx-auto px-4 py-16">
        {/* Hero Section */}
        <div className="text-center mb-16">
          <Skeleton className="w-48 h-6 mx-auto mb-4" />
          <Skeleton className="w-96 h-12 mx-auto mb-4" />
          <Skeleton className="w-96 h-12 mx-auto mb-6" />
          <Skeleton className="w-80 h-20 mx-auto mb-8" />
          <div className="flex justify-center space-x-4">
            <Skeleton className="w-32 h-11" />
            <Skeleton className="w-24 h-11" />
          </div>
        </div>

        {/* Features Grid Skeleton */}
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6 mb-16">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="w-8 h-8 mb-2" />
                <Skeleton className="w-32 h-6 mb-2" />
                <Skeleton className="w-full h-16" />
              </CardHeader>
            </Card>
          ))}
        </div>

        {/* Activity Preview Skeleton */}
        <div className="max-w-4xl mx-auto">
          <Skeleton className="w-48 h-8 mx-auto mb-12" />
          
          <Card>
            <CardHeader>
              <div className="flex items-center space-x-2">
                <Skeleton className="w-5 h-5" />
                <Skeleton className="w-32 h-6" />
                <Skeleton className="w-24 h-5" />
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex items-center space-x-3 p-3 rounded-lg bg-muted/50">
                  <Skeleton className="w-8 h-8 rounded-full" />
                  <div className="flex-1">
                    <Skeleton className="w-32 h-4 mb-1" />
                    <Skeleton className="w-48 h-3" />
                  </div>
                  <Skeleton className="w-16 h-5" />
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}