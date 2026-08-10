import { useEffect } from "react"
import { toast } from "sonner"
import { onUnauthorized } from "@/shared/api"

export function useSessionGuard(): void {
  useEffect(() => {
    return onUnauthorized(() => {
      toast.error("Session expired", {
        description: "Please sign in again to continue.",
      })
      window.location.assign("/")
    })
  }, [])
}
