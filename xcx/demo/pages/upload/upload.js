// upload.js
Page({
  data: {
    videoPath: '',  // 用于存储视频文件路径
    uploading: false, // 用于控制上传按钮的禁用状态，防止重复提交
    uploadProgress: 0, // 用于显示上传进度
  },
  chooseAndUpload: function() {
    wx.chooseMedia({
      count: 1,
      mediaType: ['video'],
      sourceType: ['album', 'camera'],
      success: (res) => {
        const tempFilePath = res.tempFiles[0].tempFilePath;
        this.setData({
          videoPath: tempFilePath,  // 将文件路径保存到 data 中
          uploadProgress: 0,  // 重置上传进度
        });
        // 可以直接上传
        this.uploadVideo();
        // // 存储选择的文件路径到本地数据
        // wx.navigateTo({
        //   url: `/pages/showmedia/showmedia?filePath=${tempFilePath}`, // 跳转到新页面并传递文件路径
        // });
      },
      fail: (err) => {
        console.log('选择文件失败', err);
      }
    });
  },
  uploadVideo: function() {
    if (!this.data.videoPath || this.data.uploading) {
      return; // 如果没有选择文件或正在上传，则不执行
    }

    this.setData({
      uploading: true, // 设置上传状态为 true，禁用上传按钮
    });

    wx.uploadFile({
      url: 'http://127.0.0.1:8080/user/upload', // 替换为你的后端上传 API 地址。 注意: 本地开发需要配置合法域名！
      filePath: this.data.videoPath,
      name: 'video', // 与 Go 代码中 c.FormFile("video") 的 name 对应
      header: {
        'content-type': 'multipart/form-data',  // 必须设置，用于上传文件
        'Authorization': wx.getStorageSync('token') // 如果需要鉴权，添加 Authorization 头
      },
      success: (res) => {
        this.setData({
          uploading: false, // 上传完成，恢复按钮状态
          uploadProgress: 0,
        });
        console.log('上传成功', res);
        const data = JSON.parse(res.data); // 解析后端返回的数据
        wx.showToast({
          title: '上传成功',
          icon: 'success',
          duration: 2000,
        });
        // 处理上传成功的逻辑，例如：显示上传结果、重置页面、跳转等
        // console.log('上传成功，服务器返回数据:', data);
      },
      fail: (err) => {
        this.setData({
          uploading: false, // 上传失败，恢复按钮状态
          uploadProgress: 0,
        });
        console.error('上传失败', err);
        wx.showToast({
          title: '上传失败',
          icon: 'error',
          duration: 2000,
        });
        // 处理上传失败的逻辑，例如：提示用户、重试等
      },
      complete: () => {
        // 不论成功与否都会执行
      },
      // 监听上传进度
      onProgressUpdate: (res) => {
        this.setData({
          uploadProgress: res.progress, // 更新上传进度百分比
        });
        console.log('上传进度', res.progress);
      }
    });
  }
});