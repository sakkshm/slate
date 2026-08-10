import { Link, Navigate, Outlet } from "react-router-dom"
import { toast } from "sonner"
import { Box, LayoutDashboard, LogOut, Loader2 } from "lucide-react"

import { useAuth } from "@/hooks/useAuth"
import { apiClient } from "@/shared/api"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { ThemeToggle } from "@/components/custom/ThemeToggle"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

function Brand() {
  return (
    <Link to="/dashboard" className="flex items-center gap-2 px-2">
      <Box className="size-5" />
      <span className="font-heading text-base font-semibold tracking-tight">
        slate
      </span>
    </Link>
  )
}

function UserMenu() {
  const { user } = useAuth()

  const handleSignOut = async () => {
    try {
      await apiClient.post("/api/auth/github/logout")
    } catch {
      toast.error("Failed to sign out, clearing session locally")
    } finally {
      window.location.href = "/"
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            className="flex items-center gap-2 rounded-full p-1 pr-2.5 outline-none transition-colors hover:bg-muted"
          >
            <Avatar size="sm">
              {user?.avatar_url ? (
                <AvatarImage src={user.avatar_url} alt={user.username} />
              ) : null}
              <AvatarFallback>
                {user?.username?.slice(0, 2).toUpperCase() ?? "?"}
              </AvatarFallback>
            </Avatar>
            <span className="hidden text-sm font-medium sm:block">
              {user?.name}
            </span>
          </button>
        }
      />
      <DropdownMenuContent align="end" className="w-56">
        <div className="px-2 py-1.5">
          <p className="truncate text-sm font-medium">{user?.name}</p>
          <p className="truncate text-xs text-muted-foreground">
            @{user?.username}
          </p>
        </div>
        <DropdownMenuItem variant="destructive" onClick={handleSignOut}>
          <LogOut />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function DashboardLayout() {
  const { loading, authed } = useAuth()

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!authed) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b bg-background px-4">
        <div className="flex min-w-0 items-center gap-4">
          <Brand />
          <Button
            variant="ghost"
            className="justify-start"
            render={
              <Link to="/dashboard">
                <LayoutDashboard data-icon="inline-start" />
                Projects
              </Link>
            }
          />
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <ThemeToggle />
          <UserMenu />
        </div>
      </header>
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
