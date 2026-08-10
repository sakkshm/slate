import { IconBrandGithub } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { apiClient, type APIError } from "@/shared/api";
import { toast } from "sonner";

export function GithubButton({ label = "Connect Github" }: { label?: string }) {
    const handleConnectGitHub = async () => {
        try {
            const data = await apiClient.get<{ url: string }>('/api/auth/github/initiate-login')
            
            // Redirect user to GitHub Auth
            window.location.href = data.url;

        } catch (err) {
            const error = err as APIError;
            console.error('Failed Status:', error.status);  
            console.error('Error Code:', error.code);      
            console.error('Message:', error.message); 
            
            toast.error(`Unable to redirect to provider: ${error.code}`, {
                description: `${error.message} (Status: ${error.status})`,
            })
        }
    };

    return (
        <div className="flex gap-2">
        <Button size="lg" onClick={handleConnectGitHub}>
            <IconBrandGithub data-icon="inline-start" /> {label}
        </Button>
        </div>
    )
}