App({
  onLaunch() {
    const settings = wx.getStorageSync("xelora_settings");
    if (!settings) {
      wx.setStorageSync("xelora_settings", {
        baseUrl: "http://localhost:8080",
        apiKey: "",
        selectedKnowledgeBaseId: ""
      });
    }
  }
});
