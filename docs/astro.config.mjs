// @ts-check
import { defineConfig } from 'astro/config'

import starlight from '@astrojs/starlight'
import tailwindcss from '@tailwindcss/vite'

import expressiveCode from 'astro-expressive-code'

import { pluginCollapsibleSections } from '@expressive-code/plugin-collapsible-sections'
import { pluginLineNumbers } from '@expressive-code/plugin-line-numbers'

// https://astro.build/config
export default defineConfig({
  integrations: [
    expressiveCode({
      themeCssSelector: (theme) => `.${theme.type}`,
      themes: ['vesper'],
      styleOverrides: {
        borderWidth: '0.1px',
      },
      frames: {
        extractFileNameFromCode: false,
        removeCommentsWhenCopyingTerminalFrames: true,
        showCopyToClipboardButton: true,
      },
      plugins: [pluginCollapsibleSections(), pluginLineNumbers()],
    }),
    starlight({
      title: 'Hyperterse',
      titleDelimiter: ' - ',
      description:
        'The interface between production databases and AI agents. Define queries once, get a high-performant engine that is reliable, interpretable, and structured.',
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
        {
          label: 'Troubleshooting',
          autogenerate: { directory: 'troubleshooting' },
        },
      ],
      components: {
        Header: '@/components/header.astro',
        PageFrame: '@/components/page-frame.astro',
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
