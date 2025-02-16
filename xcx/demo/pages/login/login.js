// pages/login/login.js
const app = getApp();
Page({
  data: {
    identifier: '',  //  用户名或邮箱
    password: '',
  },

  handleUsernameInput(e) {
    this.setData({
      identifier: e.detail.value
    });
  },

  handlePasswordInput(e) {
    this.setData({
      password: e.detail.value
    });
  },

  handleLogin() {
    const { identifier, password } = this.data;

    if (!identifier) {
      wx.showToast({
        title: '请输入用户名/邮箱',
        icon: 'none'
      });
      return;
    }
    if (!password) {
      wx.showToast({
        title: '请输入密码',
        icon: 'none'
      });
      return;
    }

    wx.request({
      url: 'http://127.0.0.1:8080/login', // 替换成你的后端登录 API
      method: 'POST',
      data: {
        identifier: identifier,  //  将 identifier 传递给后端
        password: password
      },
      header: {  // 添加 Content-Type
        'content-type': 'application/json'
      },
      success: (res) => {
        if (res.statusCode === 200) { // 登录成功，后端返回200
          if (res.data.token) { //  检查 token 是否存在
            wx.showToast({
              title: '登录成功',
              icon: 'success'
            });
            // 存储 token
            wx.setStorageSync('token', res.data.token);
            //console.log(wx.getStorageSync('token'))
            // 存储用户信息 (假设后端返回用户信息)
            if (res.data.user) {
              app.globalData.userInfo = res.data.user;
              wx.setStorage({
                key: 'userInfo',
                data: res.data.user
              });
            }


            // 跳转到首页或其他页面
            wx.navigateTo({
              url: '/pages/user/user'
              });
          }
          else {
            wx.showToast({
              title: res.data.error || '登录失败，未获取到 Token',
              icon: 'none'
            });
            console.error("登录失败，未获取到 Token", res);
          }
        } else {
          wx.showToast({
            title: res.data.error || '登录失败，请检查用户名/邮箱和密码',
            icon: 'none'
          });
          console.error("登录失败", res);
        }
      },
      fail: (err) => {
        wx.showToast({
          title: '网络错误，请稍后重试',
          icon: 'none'
        });
        console.error("登录网络错误", err);
      }
    });
  },

  navigateToRegister() {
    wx.navigateTo({
      url: '/pages/register/register'
    });
  },
})