import { IconBrandGithub } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { apiClient } from "@/shared/api";
import { toast } from "sonner";

export function GithubButton() {
    const handleConnectGitHub = async () => {
        try {
            const data = await apiClient.get<any>('/api/auth/github/initiate-login')
            
            // Redirect user to GitHub Auth
            window.location.href = data.url;

        } catch(err: any) {
            console.error('Failed Status:', err.status);  
            console.error('Error Code:', err.code);      
            console.error('Message:', err.message); 
            
            toast.error(`Unable to redirect to provider: ${err.code}`, {
                description: `${err.message} (Status: ${err.status})`,
            })
        }
    };

    return (
        <div className="flex gap-2">
        <Button onClick={handleConnectGitHub}>
            <IconBrandGithub data-icon="inline-start" /> Connect Github
        </Button>
        </div>
    )
}