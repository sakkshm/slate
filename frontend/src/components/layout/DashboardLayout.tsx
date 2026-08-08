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
            className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left outline-none transition-colors hover:bg-muted"
          >
            <Avatar size="sm">
              {user?.avatar_url ? (
                <AvatarImage src={user.avatar_url} alt={user.username} />
              ) : null}
              <AvatarFallback>
                {user?.username?.slice(0, 2).toUpperCase() ?? "?"}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1 leading-tight">
              <p className="truncate text-sm font-medium">{user?.name}</p>
              <p className="truncate text-xs text-muted-foreground">
                @{user?.username}
              </p>
            </div>
          </button>
        }
      />
      <DropdownMenuContent align="start" className="w-56">
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
    <div className="flex h-screen overflow-hidden">
      <aside className="flex w-60 shrink-0 flex-col border-r bg-sidebar">
        <div className="flex h-14 items-center justify-between border-b px-3">
          <Brand />
          <ThemeToggle />
        </div>
        <nav className="flex flex-col gap-1 p-2">
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
        </nav>
        <div className="mt-auto border-t p-2">
          <UserMenu />
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
