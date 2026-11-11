interface AppConfig {
    serverURL: string;
    priceKwh: number;
    // Location for the sunrise/sunset times in the top bar; Lisbon when unset.
    sunLat?: number;
    sunLon?: number;
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
