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
      runtimeError: null,
      connectedModel: Boolean(window.localStorage.getItem('pat_models_token_v1')),
      token: window.localStorage.getItem('pat_models_token_v1') || null,
      tokenInput: '',
      loading: false,
      executing: false,
      mode: 'fair',
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
    },
    availableModes() {
      return [
        { value: 'fair', label: 'Fair use' },
        { value: 'connected', label: 'Connected model' },
        { value: 'augmented', label: 'Augmented prompt' }
      ]
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

      let zipResponse = await fetch(`${window.location.origin}/inter/inter-font.zip`);
      let zipBinary = await zipResponse.arrayBuffer();
      this.pyodide.unpackArchive(zipBinary, 'zip', { extractDir: '/fonts' });
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
      } catch { }
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
      this.title = ''
    },
    async copyPrompt() {
      await navigator.clipboard.writeText(this.augmentedPrompt)
    },
    async remotePrompt() {
      const remoteModel = this.mode === 'fair'
        ? new RemoteModel()
        : new RemoteModel(this.token)
      this.runtimeError = null
      this.loading = true
      try {
        const { script } = await remoteModel.query(this.prompt, systemPrompt)
        this.script = script
      } catch (err) {
        this.runtimeError = new Error(`Failed prompting remote model: ${err.message}`)
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

        let dirHandle = null
        let workspaceFs = null
        if (this.isUsingWorkspace) {
          dirHandle = await showDirectoryPicker()
          const permissionStatus = await dirHandle.requestPermission({
            mode: 'readwrite',
          })
          if (permissionStatus !== 'granted') {
            throw new Error('read/write access to directory not granted')
          }
          workspaceFs = await this.pyodide.mountNativeFS(
            this.workspaceLocation,
            dirHandle,
          )
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
            startIn: dirHandle
          })
          await this.pyodide.runPython(`
            import os
            os.environ['FILE_INPUT_LOCATION'] = '${this.workspaceLocation}/${fileHandle.name}'
          `)
        }

        this.output = []
        const result = await this.pyodide.runPython(this.script)

        if (workspaceFs) {
          await workspaceFs.syncfs()
        }

        this.output.push('Script finished')
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
  <div id="container" class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-4">

    <!-- HARD RELOAD (TEMP) -->
    <div class="order-1 col-span-2 md:col-span-2 lg:col-span-1 self-center">
      <a href="https://pyla.offen.dev/">
        <h1 class="font-semibold text-2xl inline-block">
          Pyla
        </h1>
      </a>
    </div>

    <template v-if="pyodide">
    <div
      class="order-2 md:order-2 lg:order-3 col-span-2 lg:col-span-7 self-center flex flex-row justify-end items-center space-x-2">
      <Button type="outline" @click="clearAll">
        Clear form fields
      </Button>
      <a href="https://github.com/offen/pyla?tab=readme-ov-file#readme" target="_blank" rel="noopener noreferrer"
        class="ml-4 w-10 h-10 text-neutral-500 outline-2 outline-neutral-500 bg-transparent rounded-full flex items-center justify-center text-2xl font-semibold">
        ?
      </a>
    </div>

    <!-- PROMPT -->
    <div class="order-4 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6 mt-6">
      <TextArea type="lightinput" class="placeholder:text-neutral-950" placeholder="What do you want to do?"
        v-model="prompt" />
    </div>

    <!-- GENERATE SCRIPT -->
    <div class="order-5 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6">
      <div class="flex flex-col">

        <!-- labels -->
        <div class="flex flex-wrap gap-2">
          <label v-for="option in availableModes" :key="option.value"
            class="rounded-t-lg px-4 py-2 cursor-pointer transition-colors text-neutral-950"
            :class="mode === option.value ? 'bg-neutral-200' : 'bg-neutral-300'">
            <input type="radio" class="sr-only" :value="option.value" v-model="mode">
            <span class="flex items-center gap-2 my-1">
              <span class="flex h-4 w-4 items-center justify-center rounded-full outline outline-2 outline-neutral-950"
                :class="mode === option.value ? 'bg-neutral-200' : 'bg-neutral-300'">
                <span v-if="mode === option.value" class="h-2 w-2 rounded-full bg-neutral-950"></span>
              </span>
              <span class="text-sm font-medium">{{ option.label }}</span>
            </span>
          </label>
        </div>

        <!-- cards -->
        <div class="rounded-lg rounded-tl-none bg-neutral-200 p-4">

          <template v-if="mode === 'fair'">
            <p class="text-neutral-700 mt-2">
              We're offering limited free access to OpenAI GPT-4.1 for all test users.
            </p>
            <div class="w-full flex flex-row items-center justify-center mt-6">
              <Button type="fill" @click="remotePrompt" :disabled="loading">
                Generate script
              </Button>
              <span
                class="w-10 h-10 ml-2 outline-2 outline-neutral-400 bg-neutral-400 text-neutral-500 rounded-lg flex items-center justify-center">
                <template v-if="loading">
                  <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none"
                    viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                    </path>
                  </svg>
                </template>
                <template v-else></template>
              </span>
            </div>
          </template>

          <template v-else-if="mode === 'connected'">
            <div class="flex flex-col items-center">
              <div class="w-full flex flex-row items-center justify-center">
                <input v-model="tokenInput" type="text" :disabled="token"
                  :placeholder="token ? tokenDisplay : 'Paste personal access token for GitHub Models ...'"
                  class="flex-1 min-w-0 px-4 py-2 rounded-lg bg-neutral-100 text-neutral-950 focus:outline-none focus:ring-2 focus:ring-neutral-400 disabled:bg-neutral-100 disabled:text-neutral-500">
                <Button type="outline" v-if="!token" @click="provideToken" class="cursor-pointer ml-4">
                  Provide token
                </Button>
                <Button type="outline" v-if="token" @click="deleteToken" class="ml-4">
                  Disconnect
                </Button>
              </div>
              <div class="w-full flex flex-row items-center justify-center mt-6">
                <Button type="fill" @click="remotePrompt" :disabled="!token || loading">
                  Generate script
                </Button>
                <span
                  class="w-10 h-10 ml-2 outline-2 outline-neutral-400 bg-neutral-400 text-neutral-500 rounded-lg flex items-center justify-center">
                  <template v-if="loading">
                    <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none"
                      viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                      </path>
                    </svg>
                  </template>
                  <template v-else></template>
                </span>
              </div>
            </div>
          </template>

          <template v-else>
            <div class="mt-0">
              <TextArea type="light" v-model="augmentedPrompt" readonly />
            </div>
            <div class="mt-4 flex justify-center">
              <Button type="fill" @click="copyPrompt">
                Copy augmented prompt
              </Button>
            </div>
          </template>

        </div>
      </div>
    </div>

    <!-- ERROR -->
    <div v-if="globalError" class="order-6 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6">
      <div class="bg-neutral-200 rounded-lg p-4 text-red-700">
        <p class="font-mono break-words">
          {{ globalError }}
        </p>
      </div>
    </div>

    <!-- BOOKMARK -->
    <div class="order-7 col-span-2 md:col-start-1 md:col-span-4 lg:col-start-2 lg:col-span-6">
      <div class="bg-neutral-200 rounded-lg p-4">
        <div class="w-full flex flex-row items-center justify-center gap-4">
          <!-- script title become website title as well, is also saved in URL -->
          <input v-model="title" type="text" placeholder="Type script title ..."
            class="flex-1 min-w-0 px-4 py-2 rounded-lg bg-neutral-50 text-neutral-950 focus:outline-none focus:ring-2 focus:ring-neutral-400 disabled:bg-neutral-100 disabled:text-neutral-500">
          <Button type="outline" @click="saveURL">
            Generate script URL
          </Button>
        </div>
      </div>
    </div>

    <!-- RUN SCRIPT -->
    <div class="order-8 col-span-2 md:col-span-4 lg:col-span-8 mt-8">

      <div class="bg-neutral-200 rounded-lg p-4">
        <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-4">
          <div class="col-span-2 md:col-span-4 lg:col-span-8">
            <TextArea type="dark"
              :placeholder="mode=== 'augmented' ? 'Paste script from LLM ...' : 'Script goes here ...'"
              v-model="script" />
          </div>
        </div>
        <div class="w-full flex flex-row items-center justify-center mt-6">
          <Button type="fill" @click="run">
            Run script
          </Button>

          <span
            class="w-10 h-10 ml-2 outline-2 outline-neutral-400 bg-neutral-400 text-neutral-500 rounded-lg flex items-center justify-center">
            <template v-if="executing">
              <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none"
                viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                </path>
              </svg>
            </template>
            <template v-else></template>
          </span>
        </div>
      </div>
    </div>

    <!-- OUTPUT -->
    <div class="order-9 col-span-2 md:col-span-4 lg:col-span-8">
      <div class="rounded-lg p-3 bg-neutral-200 text-neutral-500">
        <pre v-if="output.length" class="break-all whitespace-pre-wrap">{{ output.join('\n') }}</pre>
        <pre v-else class="break-all whitespace-pre-wrap">Output goes here ...</pre>
      </div>
    </div>

    </template>
    <template v-else>
      <div class="order-2 col-span-2 md:col-span-4 lg:col-span-7 lg:col-start-2 flex flex-row items-center">
        <div
          class="w-10 h-10 outline-2 outline-neutral-400 bg-neutral-400 rounded-lg flex items-center justify-center ml-6 md:ml-0">
          <svg class="size-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
            </path>
          </svg>
        </div>
        <div class="h-10 text-neutral-500 flex items-center ml-6">
          <p>
            Python runtime is initializing ...
          </p>
        </div>
      </div>
    </template>

  </div>

  <footer class="mb-2 mt-20 text-neutral-500">
    <div class="flex justify-between">
      <div class="">
        November 2025
      </div>
      <div class="">
        <a href="https://github.com/offen/pyla" target="_blank" rel="noopener" class="no-underline">Source code for this
          tool</a>
      </div>
    </div>
  </footer>

</template>
