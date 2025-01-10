import { auth } from '@/lib/auth/server';
import { redirect } from 'next/navigation';
import { NextRequest } from 'next/server';

export async function GET(request: NextRequest) {
    const user = await auth.getCurrentUser();
    if (!user) {
        return redirect("/auth/sign-in");
    }
    if (!user.orgId) {
        const newOrgId = await auth.createTenant({
            name: "Personal Workspace",
            userId: user.id
        });

        // refresh session with the new orgId
        await auth.refreshSession(newOrgId);
    }
    
  
    return redirect("/new");
}