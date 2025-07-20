import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { SessionProvider } from "next-auth/react";
import { TRPCProvider } from "@/providers/trpc-provider";

const inter = localFont(
 {
  src: './fonts/opensans.ttf',
  display: 'swap',
  variable: '--font-inter',
 }
);

export const metadata: Metadata = {
  title: {
    default: "GameDev VCS",
    template: "%s | GameDev VCS"
  },
  description: "Modern version control for game development teams",
  keywords: ["version control", "game development", "collaboration", "UE5"],
  authors: [{ name: "Your Studio" }],
  creator: "Your Studio",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <ThemeProvider
          attribute="class"
          defaultTheme="dark"
          enableSystem
          disableTransitionOnChange
        >
          <SessionProvider>
            <TRPCProvider>
              {children}
            </TRPCProvider>
            <Toaster />
          </SessionProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}