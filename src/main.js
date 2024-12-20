import { createApp } from 'vue'

createApp({
  data() {
    return {
      pyodide: null,
    }
  },
  computed: {
    globalError() {
      if (this.pyodide instanceof Error) {
        return this.pyodide
      }
      return null
    }
  },
  async mounted() {
    try {
      const pyodide = await window.loadPyodide()
      this.pyodide = pyodide
    } catch (err) {
      this.pyodide = err
    }
  }
}).mount('#app')
