<script>
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'

export default {
  components: { Splitpanes, Pane },
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
      this.pyodide.setStdout({ batched: (msg) => this.output.push(msg) })
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
        await this.pyodide.runPython(this.script)

        if (this.workspaceFs) {
          await this.workspaceFs.syncfs()
        }
      } catch (err) {
        this.runtimeError = err
      }
    }
  }
}
</script>

<template>
  <div v-if="globalError">
    <p>{{ globalError.message }}</p>
    <pre>{{ globalError.stack }}</pre>
  </div>
  <div v-else-if="pyodide" id="container">
    <splitpanes horizontal class="default-theme">
      <pane max-size="20">
        <h1 class="sans-serif f1 ml4">
          Pyla
        </h1>
      </pane>
      <pane>
        <pane>
          <splitpanes horizontal>
            <pane>
              <splitpanes>
                <pane>
                  <label>
                    Script
                    <textarea class="db w-100 h-100" v-model="script"></textarea>
                  </label>
                </pane>
                <pane>
                  <label>
                    Requirements
                    <textarea class="db w-100 h-100" v-model="requirements"></textarea>
                  </label>
                </pane>
              </splitpanes>
            </pane>
            <pane max-size="42">
              <div>
                <button @click="run">Run Script</button>
              </div>
              <div>
                <pre v-if="output.length">{{ output.join('\n') }}</pre>
                <pre v-else>Output goes here ...</pre>
              </div>
            </pane>
          </splitpanes>
        </pane>
      </pane>
    </splitpanes>
  </div>
  <div v-else>
    <p>Python runtime is initializing ...</p>
  </div>
</template>
