import { createRouter, createWebHashHistory } from "vue-router";
import LoginView from "../views/LoginView.vue";
import MainView from "../views/MainView.vue";
import { getJwt, isJwtExpired } from "../utils";

const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        { path: "/", redirect: "/login" },
        { path: "/login", component: LoginView },
        { path: "/main", component: MainView, meta: { requiresAuth: true } },
    ],
});

router.beforeEach((to) => {
    if (to.meta.requiresAuth) {
        const jwt = getJwt();
        if (!jwt || isJwtExpired(jwt)) {
            return "/login";
        }
    }
});

export default router;
