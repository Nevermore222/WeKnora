import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import "./assets/fonts.css";
import TDesign from "tdesign-vue-next";
// 引入组件库的少量全局样式变量
import "tdesign-vue-next/es/style/index.css";
import "@/assets/theme/theme.css";
import "@/assets/dropdown-menu.less";
import "@/components/css/chat-hljs-dark.less";
// vue-virtual-scroller ships its own tiny stylesheet — required for
// RecycleScroller/DynamicScroller to size their viewport correctly.
// Without it the scroller computes 0 height and renders no items.
import "vue-virtual-scroller/dist/vue-virtual-scroller.css";
import i18n from "./i18n";
import { initTheme } from "@/composables/useTheme";
import { initFont } from "@/composables/useFont";
import { installTDesignIconOfflineGuard } from "@/utils/tdesign-icon-offline";
import { setDesktopBootstrap, setRuntimeContext } from "@/utils/api-context";
import { ensureDefaultDesktopProfile } from "@/api/desktop-remote";
import { initializeDefaultEnterpriseProfile } from "@/utils/default-enterprise-bootstrap";

// 必须在 Vue 组件挂载之前执行，避免 tdesign-icons 运行时请求 tdesign.gtimg.com
installTDesignIconOfflineGuard();

initTheme();
initFont();

await initDesktopBootstrap();

const app = createApp(App);

app.use(TDesign);
app.use(createPinia());
app.use(router);
app.use(i18n);

// 等首屏路由（含导航守卫、Lite 自动登录）完成后再挂载，避免先闪默认页再跳转
router.isReady().finally(() => {
  app.mount("#app");
});

async function initDesktopBootstrap() {
  const w = window as any;
  const fn = w.go?.main?.App?.GetDesktopBootstrap;
  if (typeof fn !== 'function') return;
  try {
    const bootstrap = await Promise.resolve(fn());
    setDesktopBootstrap(bootstrap);
    await initDefaultEnterpriseProfile(bootstrap);
  } catch {
    setDesktopBootstrap(null);
  }
}

async function initDefaultEnterpriseProfile(bootstrap: any) {
  try {
    await initializeDefaultEnterpriseProfile(bootstrap, {
      ensureProfile: ensureDefaultDesktopProfile,
      setContext: setRuntimeContext,
    });
  } catch (error) {
    console.error('failed to initialize default enterprise profile:', error);
  }
}
