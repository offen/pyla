import { createApp } from 'vue'

createApp({
  data() {
    return {
      pyodide: null,
      script: '',
      requirements: '',
      output: [],
      filesPath: '/home/pyodide/pyla',
      runtimeError: null,
      nativefs: null
    }
  },
  computed: {
    globalError() {
      if (this.pyodide instanceof Error) {
        return this.pyodide
      }
      if (this.runtimeError) {
        return this.runtimeError
      }
      return null
    },
    isUsingFilesystem() {
      return this.script.indexOf('FILES_PATH') !== -1
    }
  },
  async mounted() {
    try {
      const pyodide = await window.loadPyodide({
        env: {
          'FILES_PATH': this.filesPath
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
      try {
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

        if (this.isUsingFilesystem && !this.nativefs) {
          const dirHandle = await showDirectoryPicker()
          const permissionStatus = await dirHandle.requestPermission({
            mode: 'readwrite',
          })

          if (permissionStatus !== 'granted') {
            throw new Error('read access to directory not granted')
          }
          this.nativefs = await this.pyodide.mountNativeFS(this.filesPath, dirHandle)
        }

        await this.pyodide.runPython(this.script)

        if (this.nativefs) {
          await this.nativefs.syncfs()
        }
      } catch (err) {
        this.runtimeError = err
      }
    }
  }
}).mount('#app')
