<script>

import lz from 'lz-string'
import toml from 'toml'

import Button from './components/button.vue'
import TextArea from './components/textarea.vue'
import systemPrompt from './../SYSTEM_PROMPT.md?raw'
import RemoteModel from './remote-model.js'

export default {
  components: { Button, TextArea },
  data() {
    return Object.assign({
      pyodide: null,
      script: '',
      prompt: '',
      title: '',
      output: [],
      workspaceLocation: '/home/pyodide/pyla',
      workspaceFs: null,
      dirHandle: null,
      runtimeError: null,
      connectedModel: Boolean(window.localStorage.getItem('pat_models_token_v1')),
      token: window.localStorage.getItem('pat_models_token_v1') || null,
      tokenInput: '',
      loading: false,
      executing: false,
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
  watch: {
    title: {
      immediate: true,
      handler(current, prev) {
        document.title = current ? `${current} | Pyla` : 'Pyla'
      }
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
        for (const key of ['script', 'prompt', 'title']) {
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
        title: this.title,
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
      this.prompt = ''
    },
    async copyPrompt() {
      await navigator.clipboard.writeText(this.augmentedPrompt)
    },
    async remotePrompt () {
      const remoteModel = new RemoteModel(this.token)
      this.loading = true
      try {
        const { script } = await remoteModel.query(this.prompt, systemPrompt)
        this.script = script
      } catch (err) {
        this.runtimeError = new Error(`Error prompting remote model: ${err.message}`)
      } finally {
        this.loading = false
      }
    },
    parseMetadata(script) {
      const REGEX = /^# \/\/\/ ([a-zA-Z0-9-]+)$(?:\r?\n|\r)((?:^#(?: |.*)?$(?:\r?\n|\r))+)^# \/\/\/$/mg

      const name = 'script'
      const matches = []
      let match

      // Use global regex with exec to find all matches
      // Resetting lastIndex for safety, though it should be at 0 for a new search
      REGEX.lastIndex = 0
      while ((match = REGEX.exec(script)) !== null) {
          // match[1] corresponds to the 'type' named group in Python
          if (match[1] === name) {
              matches.push(match)
          }
      }

      if (matches.length > 1) {
          throw new Error(`Multiple ${name} blocks found`)
      } else if (matches.length === 1) {
        // matches[0][2] corresponds to the 'content' named group in Python
        const contentRaw = matches[0][2]

        // Process content lines: remove '# ' or '#' prefix
        const contentLines = contentRaw.split(/\r?\n|\r/);
        const processedContent = contentLines.map(line => {
            if (line.startsWith('# ')) {
                return line.substring(2)
            } else if (line.startsWith('#')) {
                return line.substring(1)
            }
            return line; // Should not happen if regex is correct, but for safety
        }).join('\n') // Join with '\n' as tomllib expects standard newlines

        try {
          return toml.parse(processedContent)
        } catch (e) {
          // Add context to the TOML parsing error
          throw new Error(`Error parsing TOML content in '${name}' block: ${e.message}`)
        }
      } else {
          return null
      }
    },
    async run() {
      try {
        this.executing = true
        const metadata = this.parseMetadata(this.script)
        if (metadata && Array.isArray(metadata.dependencies) && metadata.dependencies.length) {
          await this.pyodide.loadPackage('micropip')
              .then(() => this.pyodide.pyimport('micropip'))
              .then(async micropip => {
                for (const req of metadata.dependencies) {
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
      } finally {
        this.executing = false
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
      <p>Workspace location: <span v-if="localWorkspacePath" class="font-semibold">{{ localWorkspacePath }}</span></p>
    </div>

    <div class="order-2 md:order-2 lg:order-3 col-span-2 md:col-span-2 lg:col-span-3 self-center flex flex-row justify-end items-center space-x-2">
      <Button type="outline" @click="clearAll">
        Clear all form fields
      </Button>
      <p class="ml-4 text-neutral-500 text-2xl">
        ?
      </p>
    </div>

    <div class="order-4 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6 mt-10">
      <TextArea
        type="lightinput"
        class="placeholder:text-neutral-950"
        placeholder="What do you want to do?"
        v-model="prompt"
      />
    </div>

    <div class="order-5 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6">

      <div class="bg-neutral-200 rounded-lg p-4">
        <div class="flex items-center gap-4">
          <span>Augmented prompt</span>
          <label class="toggle-label" style="margin-bottom: 0;">
            <input type="checkbox" v-model="connectedModel" class="toggle-checkbox">
            <span class="toggle-slider"></span>
          </label>
          <span>Connected model</span>
        </div>
     
        <div v-if="connectedModel" class="bg-neutral-200 rounded-lg flex flex-col items-center">
          <div class="w-full flex flex-row items-center justify-center mt-6">
            <input
              v-model="tokenInput"
              type="text"
              :disabled="token"
              :placeholder="token ? tokenDisplay : 'Paste personal access token for GitHub Models ...'"
              class="flex-1 min-w-0 px-4 py-2 rounded-lg bg-neutral-50 text-neutral-950 focus:outline-none focus:ring-2 focus:ring-neutral-400 disabled:bg-neutral-100 disabled:text-neutral-500"
            >
            <Button type="outline" v-if="!token" @click="provideToken" class="cursor-pointer ml-4">
              Provide token
            </Button>
            <Button type="outline" v-if="token" @click="deleteToken" class="ml-4">
              Disconnect
            </Button>
          </div>
          <div class="w-full flex flex-row items-center justify-center mt-6">
            <Button
              type="fill"
              @click="remotePrompt"
              :disabled="!token"
            >
              Generate script
            </Button>
            <span class="w-10 h-10 ml-2 outline-2 outline-neutral-400 bg-neutral-400 text-neutral-500 rounded-lg flex items-center justify-center">
              <template v-if="loading">
                <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </template>
              <template v-else></template>
            </span>
          </div>
        </div>

        <template v-if="!connectedModel">
          <div class="mt-6">
            <TextArea
              type="light"
              v-model="augmentedPrompt"
              readonly
            />
          </div>
          <div class="mt-4 flex justify-center">
            <Button type="fill" @click="copyPrompt">
              Copy augmented prompt
            </Button>
          </div>
        </template>

      </div>
    </div>

    <div class="order-6 col-span-2 md:col-span-4 lg:col-span-8 mt-10">

      <div class="bg-neutral-200 rounded-lg p-4">
        <p>
          Script
        </p>
        <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-4">
          <div class="col-span-2 md:col-span-4 lg:col-span-8">
            <TextArea
              type="dark"
              :placeholder="connectedModel ? 'Script from connected model goes here ...' : 'Paste script from LLM ...'"
              v-model="script"
            />
          </div>
        </div>
        <div class="w-full flex flex-row items-center justify-center mt-6">
          <Button type="fill" @click="run">
            Run script
          </Button>
           
          <span class="w-10 h-10 ml-2 outline-2 outline-neutral-400 bg-neutral-400 text-neutral-500 rounded-lg flex items-center justify-center">
            <template v-if="executing">
              <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            </template>
            <template v-else></template>
          </span>
        </div>
      </div>
    </div>

    <div class="order-7 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6">
      <div class="bg-neutral-200 rounded-lg p-4">
        <p class="mb-2">
          Bookmark
        </p>
        <div class="w-full flex flex-row items-center justify-center gap-4">
          <!-- script title become website title as well, is also saved in URL -->
          <input
             v-model="title"
             type="text"
             placeholder="Type script title ..."
             class="flex-1 min-w-0 px-4 py-2 rounded-lg bg-neutral-50 text-neutral-950 focus:outline-none focus:ring-2 focus:ring-neutral-400 disabled:bg-neutral-100 disabled:text-neutral-500"
           >
          <Button type="outline" @click="saveURL">
            Generate script URL
          </Button>
        </div>
      </div>
    </div>

    <div class="order-8 col-span-2 md:col-span-4 lg:col-span-8 mt-10">
      <p class="mb-2">
        Output
      </p>
      <div class="rounded-lg p-3 bg-neutral-50">
          <pre v-if="output.length" class="font-mono">{{ output.join('\n') }}</pre>
          <pre v-else class="font-mono">Output goes here ...</pre>
      </div>
    </div>

  </div>

  <div v-else class="order-9 flex flex-row items-center">
      <div class="font-semibold text-2xl">
        <h1>
          Pyla
        </h1>
      </div>
      <div class="w-10 h-10 outline-2 outline-neutral-400 bg-neutral-400 rounded-lg flex items-center justify-center ml-6">
        <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>
      <div class="h-10 text-neutral-500 flex items-center ml-6">
        <p>
          Python runtime is initializing ...
        </p>
      </div>
  </div>

  <footer class="mb-2 mt-20 text-neutral-500">
    <div class="flex justify-between">
      <div class="">
        July 2025
      </div>
      <div class="">
        <a href="https://github.com/offen/pyla" target="_blank" rel="noopener" class="no-underline">Source code for this tool</a>
      </div>
    </div>
  </footer>

</template>
