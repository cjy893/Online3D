Page({
    navigateToUpload() {
        wx.navigateTo({
          url: '/pages/upload/upload',
        })
    },

    navigateToVideoSearch() {
        wx.navigateTo({
          url: '/pages/showmedia/showmedia',
        })
    },

    navigateToWorkSearch() {
        wx.navigateTo({
          url: '/pages/getwork/getwork',
        })
    },

    navigateToUserCenter() {
        wx.navigateTo({
          url: '/pages/usercenter/usercenter',
        })
    }
});