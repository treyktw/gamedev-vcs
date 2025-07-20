import { auth } from "@/auth"
import { NextResponse } from "next/server"
import { withRateLimit } from "@/lib/rate-limiting-middleware"
import { authRateLimit } from "@/lib/redis"

export default auth(async (req) => {
  const { pathname } = req.nextUrl

  // Public routes
  if (
    pathname.startsWith('/api/auth') ||
    pathname.startsWith('/auth') ||
    pathname === '/' ||
    pathname.startsWith('/_next') ||
    pathname.startsWith('/favicon')
  ) {
    return NextResponse.next()
  }

  // Protected routes
  if (pathname.startsWith('/dashboard') || pathname.startsWith('/orgs')) {
    if (!req.auth) {
      return NextResponse.redirect(new URL('/auth/signin', req.url))
    }
  }

  return NextResponse.next()
})

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)']
}
