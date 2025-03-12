<script>

import Button from './components/button.vue'

export default {
  components: { Button },
  data() {
    return {
      pyodide: null,
      script: '',
      requirements: '',
      output: [],
      workspaceLocation: '/home/pyodide/pyla',
      workspaceFs: null,
      runtimeError: null
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
    isUsingWorkspace() {
      return this.script.indexOf('WORKSPACE_LOCATION') !== -1
    },
    isUsingTextInput() {
      return this.script.indexOf('TEXT_INPUT') !== -1
    }
  },
  async mounted() {
    try {
      const pyodide = await window.loadPyodide({
        env: {
          'WORKSPACE_LOCATION': this.workspaceLocation,
        }
      })
      this.pyodide = pyodide

      this.pyodide.setStdin({ error: true })
      this.pyodide.setStdout({ batched: (msg) => this.output.push(msg) })
      this.pyodide.setStderr({ batched: (msg) => this.output.push(msg) })
    } catch (err) {
      this.pyodide = err
    }
  },
  methods: {
    async run () {
      try {
        if (this.requirements.trim()) {
          const requirements = this.requirements
            .trim()
            .split('\n')
            .filter((l) => !l.trim().startsWith('#'))

          await this.pyodide.loadPackage('micropip')
              .then(() => this.pyodide.pyimport('micropip'))
              .then(async micropip => {
                for (const req of requirements) {
                  await micropip.install(req)
                }
              })
        }

        if (this.isUsingWorkspace && !this.workspaceFs) {
          const dirHandle = await showDirectoryPicker()
          const permissionStatus = await dirHandle.requestPermission({
            mode: 'readwrite',
          })

          if (permissionStatus !== 'granted') {
            throw new Error('read/write access to directory not granted')
          }
          this.workspaceFs = await this.pyodide.mountNativeFS(this.workspaceLocation, dirHandle)
        } else if (this.workspaceFs) {
          await this.workspaceFs.syncfs()
        }

        if (this.isUsingTextInput) {
          const textInput = prompt('Provide text input here')
          await this.pyodide.runPython(`
            import os
            os.environ['TEXT_INPUT'] = '${textInput}'
          `)
        }
        this.output = []
        await this.pyodide.runPython(this.script)

        if (this.workspaceFs) {
          await this.workspaceFs.syncfs()
        }
      } catch (err) {
        this.output.push(err.message)
      }
    }
  }
}
</script>

<template>
  <div v-if="pyodide" id="container" class="max-w-256 m-auto">
    <div class="grid mb-8">
      <h1>
        Pyla
      </h1>
    </div>
    <div class="grid grid-cols-2 mb-8">
      <div>
        <label>
          Script
          <textarea class="block border-1" v-model="script"></textarea>
        </label>
      </div>
      <div>
        <label>
          Requirements
          <textarea class="block border-1" v-model="requirements"></textarea>
        </label>
      </div>
    </div>
    <div class="grid mb-8">
      <div>
        <Button @click="run">
          Run Script
        </Button>
      </div>
    </div>
    <div class="grid mb-8">
      <div>
        <pre v-if="output.length">{{ output.join('\n') }}</pre>
        <pre v-else>Output goes here ...</pre>
      </div>
    </div>
  </div>
  <div v-else>
    <p>Python runtime is initializing ...</p>
  </div>
</template>
