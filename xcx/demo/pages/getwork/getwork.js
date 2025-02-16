// index.js
Page({
    data: {
      filePath: '', // 存储文件路径
      isImage: false, // 是否为图片
      isVideo: false, // 是否为视频
    },
  
    onLoad: function(options) {
      const filePath = options.filePath;
      this.setData({
        filePath: filePath,
        isImage: filePath.endsWith('.jpg') || filePath.endsWith('.png'), // 判断是否为图片
        isVideo: filePath.endsWith('.mp4') || filePath.endsWith('.mov'), // 判断是否为视频
      });
    }
  });