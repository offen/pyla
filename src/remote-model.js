import OpenAI from 'openai'

export default class RemoteModel {
  constructor(token, model = 'openai/gpt-4.1') {
    this.client = new OpenAI({
      baseURL: 'https://models.github.ai/inference',
      apiKey: token,
      dangerouslyAllowBrowser: true
    })
    this.model = model
  }

  decorateSystemPrompt(systemPrompt) {
    return `
      ${systemPrompt}

      RESPONSE FORMAT

      Respond with JSON as specified in the given schema.
      The **script** should go in a top level "script" key, the **requirements** in a "requirements" key.
      If no external dependencies are required, return an empty string for requirements.
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
              requirements: {
                type: 'string',
                description: 'The requirements.txt as to be installed using pip',
              },
            },
            required: ['script', 'requirements'],
            additionalProperties: false,
          },
        },
      }
    })
    return JSON.parse(response.choices[0].message.content)
  }
}
