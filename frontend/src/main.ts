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

// Optional config.json (runtime); a valid serverURL there overrides BACKEND_SERVER_URL from .env (build time)
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
        serverURL: isHttpUrl(runtimeConfig.serverURL) ? runtimeConfig.serverURL! : import.meta.env.BACKEND_SERVER_URL,
    } as AppConfig;

    const app = createApp(App);

    registerECharts(app);
    app.use(router);
    app.mount("#app");
});
