import { Route, Routes } from "react-router-dom"
import { Toaster } from "@/components/ui/sonner"
import HomePage from "./pages/HomePage"
import NotFound from "./pages/NotFound"
import { GitHubCallback } from "./pages/GitHubCallback"

export default function App() {
  return (
    <>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/auth/github/callback" element={<GitHubCallback />} />

        
        <Route path="*" element={<NotFound />} />
      </Routes>
      <Toaster richColors position="bottom-right"/>
    </>
  )
}