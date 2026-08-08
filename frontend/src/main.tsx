import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from "react-router-dom";
import { ThemeProvider } from "next-themes"
import './index.css'
import App from './App'

createRoot(document.getElementById('root')!).render(
    <BrowserRouter>
        <StrictMode>
            <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
                <App />
            </ThemeProvider>
        </StrictMode>
    </BrowserRouter>
)
