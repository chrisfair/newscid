import {createApp} from 'vue'
import '@fontsource-variable/inter'
import '@fontsource-variable/jetbrains-mono'
import App from './App.vue'
import './style.css'
import { initWailsShim } from './wails-shim'

initWailsShim()
createApp(App).mount('#app')
