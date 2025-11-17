import OpenAI from 'openai'

const FAIR_USE_TOKEN = 'pyla-fair-use'

export default class RemoteModel {
  constructor(token = FAIR_USE_TOKEN, model = 'openai/gpt-4.1-mini') {
    this.client = new OpenAI({
      baseURL: `${window.location.origin}/inference`,
      apiKey: token,
      dangerouslyAllowBrowser: true,
      timeout: 30 * 1000,
    })
    this.model = model
  }

  decorateSystemPrompt(systemPrompt) {
    return `
      ${systemPrompt}

      RESPONSE FORMAT

      Respond with JSON as specified in the given schema.
      The **script** content goes in a top level "script" key.
    `
  }

  async query(userPrompt, systemPrompt = '') {
    const response = await this.client.chat.completions.create({
      messages: [
        { role: 'system', content: this.decorateSystemPrompt(systemPrompt) },
        { role: 'user', content: userPrompt }
      ],
      temperature: 1,
      top_p: 1,
      model: this.model,
      response_format: {
        type: 'json_schema',
        json_schema: {
          strict: true,
          name: 'pyla',
          schema: {
            type: 'object',
            properties: {
              script: {
                type: 'string',
                description: 'The Python script to be run in the Pyodide runtime',
              },
            },
            required: ['script'],
            additionalProperties: false,
          },
        },
      }
    })
    return JSON.parse(response.choices[0].message.content)
  }
}
