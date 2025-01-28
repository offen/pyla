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
      filesInputLocation: '/home/pyodide/pyla/input',
      filesOutputLocation: '/home/pyodide/pyla/output',
      runtimeError: null,
      inputFs: null,
      outputFs: null,
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
    isUsingFilesystemInput() {
      return this.script.indexOf('FILES_INPUT_LOCATION') !== -1
    },
    isUsingFilesystemOutput() {
      return this.script.indexOf('FILES_OUTPUT_LOCATION') !== -1
    },
    isUsingTextInput() {
      return this.script.indexOf('TEXT_INPUT') !== -1
    }
  },
  async mounted() {
    try {
      const pyodide = await window.loadPyodide({
        env: {
          'FILES_INPUT_LOCATION': this.filesInputLocation,
          'FILES_OUTPUT_LOCATION': this.filesOutputLocation,
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

        if (this.isUsingFilesystemInput && !this.inputFs) {
          const dirHandle = await showDirectoryPicker()
          const permissionStatus = await dirHandle.requestPermission({
            mode: 'read',
          })

          if (permissionStatus !== 'granted') {
            throw new Error('read access to directory not granted')
          }
          this.inputFs = await this.pyodide.mountNativeFS(this.filesInputLocation, dirHandle)
        } else if (this.inputFs) {
          await this.inputFs.syncfs()
        }

        if (this.isUsingFilesystemOutput && !this.outputFs) {
          const dirHandle = await showDirectoryPicker()
          const permissionStatus = await dirHandle.requestPermission({
            mode: 'readwrite',
          })

          if (permissionStatus !== 'granted') {
            throw new Error('read/write access to directory not granted')
          }
          this.outputFs = await this.pyodide.mountNativeFS(this.filesOutputLocation, dirHandle)
        } else if (this.outputFs) {
          await this.outputFs.syncfs()
        }

        if (this.isUsingTextInput) {
          const textInput = prompt('Provide text input here')
          await this.pyodide.runPython(`
            import os
            os.environ['TEXT_INPUT'] = '${textInput}'
          `)
        }

        await this.pyodide.runPython(this.script)

        if (this.outputFs) {
          await this.outputFs.syncfs()
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
        <splitpanes>
          <pane max-size="25">
            File Tree goes here ...
          </pane>
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
        </splitpanes>
      </pane>
    </splitpanes>
  </div>
  <div v-else>
    <p>Python runtime is initializing ...</p>
  </div>
</template>
