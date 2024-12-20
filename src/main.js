import { createApp } from 'vue'

createApp({
  data() {
    return {
      pyodide: null,
      script: '',
      requirements: '',
      output: ['Output goes here ...']
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
      const pyodide = await window.loadPyodide({
        env: {
          'FILES_PATH': '/home/pyodide/pyla'
        }
      })
      this.pyodide = pyodide
      this.pyodide.setStdout({ batched: (msg) => this.output.push(msg) })
    } catch (err) {
      this.pyodide = err
    }
  },
  methods: {
    async run () {
      if (this.requirements.trim()) {
        const requirements = this.requirements.trim().split('\n')
        await this.pyodide.loadPackage('micropip')
            .then(() => this.pyodide.pyimport('micropip'))
            .then(async micropip => {
              for (const req of requirements) {
                await micropip.install(req)
              }
            })
      }
      const dirHandle = await showDirectoryPicker()
      const permissionStatus = await dirHandle.requestPermission({
        mode: 'readwrite',
      })

      if (permissionStatus !== 'granted') {
        throw new Error('read access to directory not granted')
      }
      const nativefs = await this.pyodide.mountNativeFS('/home/pyodide/pyla', dirHandle)
      await this.pyodide.runPython(this.script)
    }
  }
}).mount('#app')
