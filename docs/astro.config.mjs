// @ts-check
import { defineConfig } from 'astro/config'

import starlight from '@astrojs/starlight'
import tailwindcss from '@tailwindcss/vite'

import expressiveCode from 'astro-expressive-code'

// import { pluginCollapsibleSections } from '@expressive-code/plugin-collapsible-sections'
import { pluginLineNumbers } from '@expressive-code/plugin-line-numbers'

const TITLE = 'Hyperterse - The Production Database Interface for AI Agents'
const DESCRIPTION = 'Connect AI agents to production databases safely. Define queries once with Hyperterse for a high-performance, reliable, and structured SQL engine. Stop hallucinations.'
const KEYWORDS = 'Hyperterse, AI agents, database interface, SQL engine, PostgreSQL, MySQL, Redis, REST API, MCP, Model Context Protocol, LLM integration, RAG, AI tools, database queries, production database, OpenAPI, type-safe queries, database gateway, AI-first, chatbot, multi-agent systems, Docker, Kubernetes, AWS, Azure, GCP, Vercel, Railway, DigitalOcean, Cloudflare, database security, query validation, configuration-driven API'
const OG_IMAGE = '/og.png'


// https://astro.build/config
export default defineConfig({
  site: 'https://docs.hyperterse.com',
  integrations: [
    expressiveCode({
      themes: ['vesper'],
      styleOverrides: {
        borderWidth: '0.5px',
        frames: {
          editorTabBarBackground: 'var(--color-surface)',
          editorActiveTabBackground: 'var(--color-surface)',
          editorBackground: 'var(--color-surface)',
          terminalBackground: 'var(--color-surface)',
          terminalTitlebarBackground: 'var(--color-surface)',
        }
      },
      frames: {
        extractFileNameFromCode: false,
        removeCommentsWhenCopyingTerminalFrames: true,
        showCopyToClipboardButton: true,
      },
      plugins: [pluginLineNumbers()],
      defaultProps: {
        showLineNumbers: true,
        overridesByLang: {
          'bash,sh,shell,text': {
            showLineNumbers: false,
          },
        },
      }
    }),
    starlight({
      title: 'Hyperterse - The Production Database Interface for AI Agents',
      titleDelimiter: ' - ',
      description:
        'Connect AI agents to production databases safely. Define queries once with Hyperterse for a high-performance, reliable, and structured SQL engine. Stop hallucinations.',
      head: [
        {
          tag: 'meta',
          attrs: {
            property: 'og:title',
            content: TITLE,
          },
        },
        {
          tag: 'meta',
          attrs: {
            property: 'og:description',
            content: DESCRIPTION,
          },
        },
        {
          tag: 'meta',
          attrs: {
            property: 'og:image',
            content: OG_IMAGE,
          },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'twitter:card',
            content: 'summary_large_image',
          },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'twitter:title',
            content: TITLE,
          },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'twitter:description',
            content: DESCRIPTION,
          },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'twitter:image',
            content: OG_IMAGE,
          },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'keywords',
            content: KEYWORDS,
          },
        },
      ],
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/hyperterse/hyperterse',
        },
      ],
      sidebar: [
        'index',
        {
          label: 'Getting started',
          autogenerate: { directory: 'getting-started' },
        },
        {
          label: 'Concepts',
          autogenerate: { directory: 'concepts' },
        },
        {
          label: 'Learn',
          autogenerate: { directory: 'guides' },
        },
        {
          label: 'Databases',
          autogenerate: { directory: 'databases' },
        },
        {
          label: 'Deployment',
          items: [
            'deployment',
            {
              label: 'Methods',
              autogenerate: { directory: 'deployment/how' },
            },
            {
              label: 'Providers',
              autogenerate: { directory: 'deployment/where' },
            },
          ],
        },
        {
          label: 'Reference',
          autogenerate: { directory: 'reference' },
        },
        {
          label: 'Security',
          autogenerate: { directory: 'security' },
        },
      ],
      components: {
        ThemeSelect: '@/components/empty-component.astro',
        ThemeProvider: '@/components/force-dark-theme.astro',
        Header: '@/components/header.astro',
        PageFrame: '@/components/page-frame.astro',
        Search: '@/components/search.astro',
      },
      customCss: ['./src/styles/global.css'],
      pagination: true,
      prerender: true,
    }),
  ],
  vite: {
    plugins: [tailwindcss()],
  },
})
