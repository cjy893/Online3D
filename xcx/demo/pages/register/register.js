// pages/register/register.js
const app = getApp();

Page({
  data: {
    username: '',
    email: '',
    password: '',
    confirmPassword: ''
  },

  handleUsernameInput(e) {
    this.setData({ username: e.detail.value });
  },

  handleEmailInput(e) {
    this.setData({ email: e.detail.value });
  },

  handlePasswordInput(e) {
    this.setData({ password: e.detail.value });
  },

  handleConfirmPasswordInput(e) {
    this.setData({ confirmPassword: e.detail.value });
  },

  registerUser(data) { //  注册用户函数
    return new Promise((resolve, reject) => {
      wx.request({
        url: 'http://127.0.0.1:8080/register', //  替换成你的后端注册 API
        method: 'POST',
        data: data,
        header: { //  添加 Content-Type
          'content-type': 'application/json'
        },
        success: (res) => {
          if (res.statusCode === 201) { // 注册成功状态码是 201
            resolve({ success: true, message: res.data.message, data: res.data });
          } else {
            reject({ success: false, message: res.data.error || '注册失败', statusCode: res.statusCode });
          }
        },
        fail: (err) => {
          reject({ success: false, message: '网络错误，请稍后重试', error: err });
        },
      });
    });
  },

  register() {
    const { username, email, password, confirmPassword } = this.data;

    if (!username || !email || !password || !confirmPassword) {
      wx.showToast({ title: '请填写完整信息', icon: 'error' });
      return;
    }

    if (password !== confirmPassword) {
      wx.showToast({ title: '两次密码不一致', icon: 'error' });
      return;
    }

    // 调用后端注册接口
    this.registerUser({
      username,
      email,
      password
    })
    .then(res => {
        if (res.success) {
          wx.showToast({ title: '注册成功', icon: 'success' });
          // 可在此处跳转到登录页面或其他页面
          wx.navigateTo({
            url: '/pages/login/login' // 假设有登录页面
          });
        } else {
          wx.showToast({ title: res.message || '注册失败', icon: 'error' });
        }
      })
    .catch(err => {
        console.error('注册失败:', err);
        wx.showToast({ title: err.message || '注册失败，请稍后重试', icon: 'error' });
      });
  },

  goToLogin(){
    wx.navigateTo({
      url: '/pages/login/login'
    });
  }
});