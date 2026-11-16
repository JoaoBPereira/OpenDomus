interface AppConfig {
    serverURL: string;
    priceKwh: number;
    // Location for the sunrise/sunset times in the top bar; Lisbon when unset.
    sunLat?: number;
    sunLon?: number;
    // Google Maps JavaScript API key for the car route; without it the map shows only the latest position.
    googleMapsApiKey?: string;
}

declare var Config: AppConfig;

// Allow importing image and asset files in TypeScript
declare module '*.png' {
	const src: string;
	export default src;
}
declare module '*.jpg' {
	const src: string;
	export default src;
}
declare module '*.svg' {
	const src: string;
	export default src;
}
declare module '*.gif' {
	const src: string;
	export default src;
}
