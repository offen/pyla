<script>

import Button from './components/button.vue'
import TextArea from './components/textarea.vue'
import systemPrompt from './../SYSTEM_PROMPT.md?raw'

export default {
  components: { Button, TextArea },
  data() {
    const urlState = {}
    try {
      const urlData = JSON.parse(window.atob(window.location.hash.replace(/^#/, '')))
      for (const key of ['script', 'requirements', 'prompt']) {
        if (typeof urlData[key] === 'string') {
          urlState[key] = urlData[key]
        }
      }
    } catch {}

    return Object.assign({
      pyodide: null,
      script: '',
      requirements: '',
      prompt: '',
      output: [],
      workspaceLocation: '/home/pyodide/pyla',
      workspaceFs: null,
      dirHandle: null,
      runtimeError: null
    }, urlState)
  },
  computed: {
    augmentedPrompt() {
      return systemPrompt + '\n' + this.prompt + '\n'
    },
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
    },
    localWorkspacePath() {
      return this.dirHandle ? this.dirHandle.name : null
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
    saveURL() {
      const state = JSON.stringify({
        prompt: this.prompt,
        code: this.code,
        requirements: this.requirements
      })
      window.location.hash = window.btoa(state)
    },
    async copyPrompt() {
      await navigator.clipboard.writeText(this.augmentedPrompt)
    },
    async run() {
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
          this.dirHandle = dirHandle
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
        const result = await this.pyodide.runPython(this.script)

        this.output.push('Script finished')

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

  <div v-if="pyodide" id="container" class="max-w-1024 grid grid-cols-8 gap-4">

    <div class="col-span-2">
      <h1>
        Pyla
      </h1>
    </div>

    <div class="col-span-5">
      <p>Workspace Location: <span v-if="localWorkspacePath">{{ localWorkspacePath }}</span></p>
    </div>

    <div class="">
      <p>?</p>
    </div>

    <div class="col-start-2 col-span-6">
      <TextArea
        label="Prompt"
        v-model="prompt"
      />
    </div>
    
    <div class="col-start-2 col-span-6">
      <TextArea
        v-model="augmentedPrompt"
        readonly
      />
    </div>

    <div class="col-start-6 col-span-2">
        <Button @click="copyPrompt">
          Copy augmented prompt
        </Button>
    </div>

    <div class="col-start-1 col-span-8">
      <TextArea
        label="Script"
        v-model="script"
      />
    </div>

    <div class="col-start-1 col-span-3">
      <TextArea
        label="Requirements"
        v-model="requirements"
      />
    </div>

    <div class="col-start-7 col-span-2">
      <div class="flex flex-col">
        <Button @click="run">
          Run Script
        </Button>

        <Button @click="saveURL">
          Save URL
        </Button>
      </div>
    </div>

    <div class="col-start-1 col-span-8">
        <pre v-if="output.length">{{ output.join('\n') }}</pre>
        <pre v-else>Output goes here ...</pre>
    </div>

  </div>

  <div v-else>
    <p>Python runtime is initializing ...</p>
  </div>

</template>
