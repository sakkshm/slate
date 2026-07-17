import { apiClient } from "@/shared/api"
import { useEffect, useState } from "react"

function DashboardPage() {

    const [userData, setUserData] = useState("");

    useEffect(() => {
        apiClient.get("/api/user/repos").then((res) => {
            setUserData(JSON.stringify(res));
        })
    }, [])

  return (
    <div>
        <p>Dashboard Page</p>
        <p>{userData}</p>
    </div>
  )
}

export default DashboardPage