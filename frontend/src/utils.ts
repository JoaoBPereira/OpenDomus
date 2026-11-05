import { getLatestTagValueFloat } from "./services/api";

// Helper function to check JWT expiration
// "Keep me signed in" stores the token in localStorage, otherwise in sessionStorage.
export function getJwt(): string | null {
        return localStorage.getItem("jwt") ?? sessionStorage.getItem("jwt");
}

export function clearJwt(): void {
        localStorage.removeItem("jwt");
        sessionStorage.removeItem("jwt");
}

export function isJwtExpired(token: string): boolean {
        try {
            const payload = JSON.parse(atob(token.split(".")[1]));
            // exp is in seconds since epoch
            if (!payload.exp) return true;
            const now = Math.floor(Date.now() / 1000);
            return payload.exp < now;
        } catch {
            return true;
        }
}

export class FloatTimeValue {
  value: number = 0;
  timestamp: Date = new Date();
}

export const getLatestFloat = async (tagId: string): Promise<FloatTimeValue> => {
        let value = { value: 0, timestamp: new Date() };
        await getLatestTagValueFloat(tagId)
            .then((json) => {
                value = { value: json.value, timestamp: new Date(Date.parse(json.timestamp ?? json.time)) };
            });
        return value; // Default return value
    };


