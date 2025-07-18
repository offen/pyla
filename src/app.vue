<script>

import lz from 'lz-string'

import ButtonMain from './components/buttonmain.vue'
import ButtonSub from './components/buttonsub.vue'
import TextAreaLightInput from './components/textarealightinput.vue'
import TextAreaLight from './components/textarealight.vue'
import TextAreaDark from './components/textareadark.vue'
import systemPrompt from './../SYSTEM_PROMPT.md?raw'
import RemoteModel from './remote-model.js'

export default {
  components: { ButtonMain, ButtonSub, TextAreaLightInput, TextAreaLight, TextAreaDark },
  data() {
    return Object.assign({
      pyodide: null,
      script: '',
      requirements: '',
      prompt: '',
      output: [],
      workspaceLocation: '/home/pyodide/pyla',
      workspaceFs: null,
      dirHandle: null,
      runtimeError: null,
      connectedModel: Boolean(window.localStorage.getItem('pat_models_token_v1')),
      token: window.localStorage.getItem('pat_models_token_v1') || null,
      tokenInput: '',
      loading: false,
    }, this.parseUrlState())
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
    isUsingFileInput() {
      return this.script.indexOf('FILE_INPUT_LOCATION') !== -1
    },
    isUsingTextInput() {
      return this.script.indexOf('TEXT_INPUT') !== -1
    },
    localWorkspacePath() {
      return this.dirHandle ? this.dirHandle.name : null
    },
    tokenDisplay() {
      return this.token
        ? `${this.token.substr(0, 3)}...${this.token.slice(-3)}`
        : null
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
    window.addEventListener('hashchange', () => this.handleHashChange())
  },
  methods: {
    parseUrlState() {
      const urlState = {}
      try {
        const urlData = JSON.parse(lz.decompressFromEncodedURIComponent(window.location.hash.replace(/^#/, '')))
        for (const key of ['script', 'requirements', 'prompt']) {
          if (typeof urlData[key] === 'string') {
            urlState[key] = urlData[key]
          }
        }
      } catch {}
      return urlState
    },
    handleHashChange() {
      const urlState = this.parseUrlState()
      Object.assign(this.$data, urlState)
    },
    saveURL() {
      const state = JSON.stringify({
        prompt: this.prompt,
        script: this.script,
        requirements: this.requirements
      })
      window.location.hash = lz.compressToEncodedURIComponent(state)
    },
    provideToken() {
      this.token = this.tokenInput
      window.localStorage.setItem('pat_models_token_v1', this.token)
      this.tokenInput = ''
    },
    deleteToken() {
      this.token = null
      window.localStorage.removeItem('pat_models_token_v1')
    },
    clearAll() {
      this.script = ''
      this.requirements = ''
      this.prompt = ''
    },
    async copyPrompt() {
      await navigator.clipboard.writeText(this.augmentedPrompt)
    },
    async remotePrompt () {
      const remoteModel = new RemoteModel(this.token)
      this.loading = true
      try {
        const { script, requirements } = await remoteModel.query(this.prompt, systemPrompt)
        this.script = script
        this.requirements = requirements
      } catch (err) {
        this.runtimeError = new Error(`Error prompting remote model: ${err.message}`)
      } finally {
        this.loading = false
      }
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
        if (this.isUsingFileInput) {
          const [fileHandle] = await showOpenFilePicker({
            startIn: this.dirHandle
          })
          await this.pyodide.runPython(`
            import os
            os.environ['FILE_INPUT_LOCATION'] = '${this.workspaceLocation}/${fileHandle.name}'
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

  <div v-if="pyodide" id="container" class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-4">

    <div class="order-1 col-span-2 md:col-span-2 lg:col-span-1 self-center font-semibold text-2xl">
      <h1>
        Pyla
      </h1>
    </div>

    <div class="order-3 md:order-3 lg:order-2 col-span-2 md:col-span-4 lg:col-span-4 self-center text-neutral-500 bg-neutral-200 rounded-lg px-4 py-2 inline-flex w-fit">
      <p>Workspace location: <span v-if="localWorkspacePath">{{ localWorkspacePath }}</span></p>
    </div>

    <div class="order-2 md:order-2 lg:order-3 col-span-2 md:col-span-2 lg:col-span-3 self-center flex flex-row justify-end items-center space-x-2">
      <ButtonSub @click="clearAll">
        Clear all form fields
      </ButtonSub>
      <p class="ml-4 text-neutral-500 text-2xl">
        ?
      </p>
    </div>

    <div class="order-4 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6 mt-10">
      <TextAreaLightInput
        label="Tool"
        placeholder="What do you want to do?"
        v-model="prompt"
      />
    </div>

    <div class="order-5 col-span-2 md:col-span-4 lg:col-start-5 lg:col-span-3 flex justify-center md:justify-end">
      <label>
        Use Connected Model:
        <input type="checkbox" v-model="connectedModel">
      </label>
    </div>

    <div v-if="connectedModel" class="order-5 col-span-2 md:col-span-4 lg:col-start-5 lg:col-span-3 flex justify-center md:justify-end">
      <input
        v-model="tokenInput"
        type="text"
        :disabled="token"
        :placeholder="token ? tokenDisplay : 'Paste personal access token for GitHub Models'"
      >
      <ButtonMain v-if="!token" @click="provideToken" class="cursor-pointer">
        Provide token
      </ButtonMain>
      <ButtonMain v-if="token" @click="deleteToken">
        Disconnect
      </ButtonMain>
      <ButtonMain
        @click="remotePrompt"
        :disabled="!token"
      >
        Generate script via connected model
      </ButtonMain>
      <span>
        <template v-if="loading">
          Loading ...
        </template>
        <template v-else>
          ...
        </template>
      </span>
    </div>
    
    <template v-if="!connectedModel">
      <div class="order-6 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6">
        <TextAreaLight
          label="Augmented prompt"
          v-model="augmentedPrompt"
          readonly
        />
      </div>

      <div class="order-7 col-span-2 md:col-span-4 lg:col-start-5 lg:col-span-3 flex justify-center md:justify-end">
        <ButtonMain @click="copyPrompt">
          Copy augmented prompt
        </ButtonMain>
      </div>
    </template>

    <div class="order-7 col-span-2 md:col-span-4 lg:col-span-8 mt-10">
      <TextAreaDark
        label="Script"
        placeholder="Paste script from LLM ..."
        v-model="script"
      />
    </div>

    <div class="order-8 col-span-2 md:col-span-2 lg:col-span-3">
      <TextAreaDark
        label="Requirements"
        placeholder="Paste requirements from LLM ..."
        v-model="requirements"
      />
    </div>

    <div class="order-9 col-span-2 md:col-start-3 md:col-span-2 lg:col-start-6 lg:col-span-3 flex justify-center md:justify-end">
      <div class="flex flex-col self-end">
        <ButtonMain @click="run" class="mb-4">
          Run script
        </ButtonMain>

        <ButtonSub @click="saveURL">
          Generate script URL
        </ButtonSub>
      </div>
    </div>

    <div class="order-10 col-span-2 md:col-span-4 lg:col-span-8 mt-10 mb-20">
      <p class="block mb-2 text-sm/5 text-neutral-500">
          Output
      </p>
      <div class="rounded-lg p-3 bg-neutral-50">
          <pre v-if="output.length" class="font-mono">{{ output.join('\n') }}</pre>
          <pre v-else class="font-mono">Output goes here ...</pre>
      </div>
    </div>

  </div>

  <div v-else class="order-11 col-span-2 md:col-span-4 lg:col-span-8 text-neutral-500">
    <p>Python runtime is initializing ...</p>
  </div>

</template>
