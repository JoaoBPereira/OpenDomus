/**
 * Centralized API service for all HTTP requests
 * All API calls to the backend server should go through this service
 */

import router from "../router";
import { clearJwt, getJwt } from "../utils";

// Helper function to get auth headers
const getAuthHeaders = () => ({
    "Content-Type": "application/json",
    Authorization: `Bearer ${getJwt()}`,
});

// Clears the session and sends the user to the login page on 401
const handleUnauthorized = (response: Response) => {
    if (response.status !== 401) return;
    clearJwt();
    if (router.currentRoute.value.path !== "/login") {
        router.push("/login");
    }
};

// Helper function to make authenticated GET requests
const authenticatedFetch = async <T>(
    endpoint: string,
    method = "GET",
    body?: Record<string, unknown>,
): Promise<T> => {
    const response = await fetch(Config.serverURL + endpoint, {
        method,
        headers: getAuthHeaders(),
        ...(body && { body: JSON.stringify(body) }),
    });
    handleUnauthorized(response);
    if (!response.ok) {
        throw new Error(`API error: ${response.status} ${response.statusText}`);
    }
    return response.json() as Promise<T>;
};

// ============= Authentication APIs =============

export const login = async (username: string, password: string) => {
    const response = await fetch(`${Config.serverURL}login`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, password }),
    });
    return response;
};

// ============= Tag Value APIs =============

export const getTags = async (): Promise<Tag[]> => {
    return authenticatedFetch<Tag[]>(`getTags`);
};

export const getLatestTagValueFloat = async (tagId: number | string): Promise<TagValueFloat> => {
    return authenticatedFetch<TagValueFloat>(`latestTagValueFloat/${tagId}`);
};

export const getLastTagValueFloat = async (tagId: number | string, count: number): Promise<TagValueFloat[]> => {
    return authenticatedFetch<TagValueFloat[]>(`lastTagValueFloat/${tagId}/${count}`);
};

export const getLastTagValueBoolean = async (tagId: number | string, count: number): Promise<TagValueBoolean[]> => {
    return authenticatedFetch<TagValueBoolean[]>(`lastTagValueBoolean/${tagId}/${count}`);
};

export const getTagValueFloat = async (
    tagId: number | string,
    startDate: string,
    endDate: string
): Promise<TagValueFloat[]> => {
    return authenticatedFetch<TagValueFloat[]>(`tagValueFloat/${tagId}/${startDate}/${endDate}`);
};

export const getTagValueFloatTimeSeries = async (
    tagId: number | string,
    startDate: string,
    endDate: string,
    interval: number
): Promise<TagValueTimeSeries[]> => {
    return authenticatedFetch<TagValueTimeSeries[]>(`tagValueFloatTimeSeries/${tagId}/${startDate}/${endDate}/${interval}`);
};

// ============= Power Consumption APIs =============

export const getPowerConsumption = async (
    tagId: number | string,
    startDate: string,
    endDate: string
): Promise<number> => {
    return authenticatedFetch<number>(`powerconsumption/${tagId}/${startDate}/${endDate}`);
};

// ============= Power Consumption Near Timestamp APIs =============

export const getPowerConsumptionNearTimestamp = async (
    timestamp: string
): Promise<PowerConsumptionNearTimestamp[]> => {
    return authenticatedFetch<PowerConsumptionNearTimestamp[]>(`powerConsumptionNearTimestamp/${timestamp}`);
};

// ============= Server APIs =============

export const getServerVersion = async (): Promise<string> => {
    const response = await fetch(Config.serverURL);
    if (!response.ok) {
        throw new Error(`API error: ${response.status} ${response.statusText}`);
    }
    const text = await response.text();
    return text.match(/v([\d.]+)/)?.[1] ?? text;
};

// ============= Weather APIs =============

export interface WeatherForecastPeriod {
    time: string;
    periodHours: number;
    temperature: number;
    windSpeed: number;
    windDirection: string;
    precipitationProbability: number | null;
    weatherTypeId: number;
}

export interface WeatherLocation {
    district: string;
    local: string;
    globalIdLocal: number;
    latitude: number;
    longitude: number;
    idAreaAviso: string;
}

export interface WeatherDailyForecast {
    date: string;
    temperatureMin: number;
    temperatureMax: number;
    precipitationProbability: number;
    weatherTypeId: number;
    windDirection: string;
}

export interface WeatherHourly24 {
    entries: WeatherForecastPeriod[];
    temperatureMin: number;
    temperatureMax: number;
}

export interface WeatherResponse {
    sourceUrl: string;
    location: WeatherLocation;
    daily10: WeatherDailyForecast[];
    hourly24: WeatherHourly24;
    lastUpdated: string;
}

export const getWeather = async (): Promise<WeatherResponse> => {
    return authenticatedFetch<WeatherResponse>("weather/ipma");
};

// ============= Helper Types =============

export interface Tag {
    id: number;
    name: string;
    type: string;
    valueType: string;
    valueStorage: string;
    config: any;
    deviceId: number;
    periodicityCount: number;
    expiresIn: number;
    enabled: boolean;
    lastTagValue: TagValueFloat | TagValueBoolean;
    displayIn: string[];
}

export interface TagValueFloat {
    value: number;
    time: string;
    timestamp?: string;
}

export interface TagValueBoolean {
    value: boolean;
    time: string;
}

export interface TagValueTimeSeries {
    bucket: string;
    avg_value: number;
    min_value: number;
    max_value: number;
}


export interface PowerConsumptionNearTimestamp {
    tag: Tag;
    value: number | null;
    time: string;
}

