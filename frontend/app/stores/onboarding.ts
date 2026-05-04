import { defineStore } from 'pinia'

export const useOnboardingStore = defineStore('onboarding', {
  state: () => ({
    appId: '',
    appSecret: '',
    accessToken: '',
  }),
  actions: {
    reset() {
      this.appId = ''
      this.appSecret = ''
      this.accessToken = ''
    },
  },
})
