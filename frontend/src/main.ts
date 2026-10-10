import {createApp} from 'vue'
import App from './App.vue'
import './style.css'
import { initWailsShim } from './wails-shim'

initWailsShim()
createApp(App).mount('#app')
