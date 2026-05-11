import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerSW } from 'virtual:pwa-register'
import PrimeVue from 'primevue/config'

import './assets/tailwind.css'
import App from './App.vue'
import router from './router'

registerSW({ immediate: true })

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(PrimeVue, { unstyled: true })
app.mount('#app')
