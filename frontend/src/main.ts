import { createApp } from "vue";
import App from "./App.vue";

declare const __APP_VERSION__: string;
import router from "./router";
import { registerECharts } from "./plugins/echarts";
import "./index.css";

console.log(`OD v${__APP_VERSION__}`);

const isHttpUrl = (url?: string) => {
    try {
        return ["http:", "https:"].includes(new URL(url ?? "").protocol);
    } catch {
        return false;
    }
};

// The backend URL comes from config.json, read at runtime
const loadRuntimeConfig = async (): Promise<Partial<AppConfig>> => {
    try {
        const response = await fetch("config.json", { cache: "no-store" });
        return response.ok ? await response.json() : {};
    } catch {
        return {};
    }
};

loadRuntimeConfig().then((runtimeConfig) => {
    globalThis.Config = {
        ...runtimeConfig,
        serverURL: isHttpUrl(runtimeConfig.serverURL)
            ? runtimeConfig.serverURL!
            // no config.json or no valid URL in it: the API is on port 4000 of the same host
            : `${location.protocol}//${location.hostname}:4000/`,
    } as AppConfig;

    const app = createApp(App);

    registerECharts(app);
    app.use(router);
    app.mount("#app");
});
