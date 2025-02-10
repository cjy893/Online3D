// app.js
App({
  globalData: {
    userInfo: null, // 初始化 userInfo
    // ... 其他全局数据 ...
  },
  onLaunch: function () {
    //  可以从本地存储中读取 userInfo (可选)
    const userInfo = wx.getStorageSync('userInfo');
    if (userInfo) {
      this.globalData.userInfo = userInfo;
    }
  },
})
