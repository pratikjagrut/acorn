// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Acorn Docs',
  tagline: 'Welcome to Acorn Docs',
  url: 'http://docs.acorn.io',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  trailingSlash: false,
  onBrokenMarkdownLinks: 'warn',
  onDuplicateRoutes: 'warn',
  favicon: 'img/favicon.png',
  organizationName: 'acorn-io',
  projectName: 'acorn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          versions: {
			"0.6": {label: "0.6", banner: "none", path: "0.6"},
			"0.5": {label: "0.5", banner: "none", path: "0.5"},
          },
          routeBasePath: '/', // Serve the docs at the site's root
          sidebarPath: require.resolve('./sidebars.js'),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/acorn-io/runtime/tree/main/docs/',
        },
        blog: false,
        gtag: {
          trackingID: 'G-B0PL797F38',
          anonymizeIP: true,
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'Docs',
        style: 'dark',
        logo: {
          alt: 'Acorn Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            to: 'https://www.acorn.io',
            label: 'Acorn Home',
            position: 'right',
            target: '_self',
          },
          {
            to: 'https://github.com/acorn-io/runtime',
            label: 'GitHub',
            position: 'right',
          },
          {
            type: 'docsVersionDropdown',
            position: 'left',
            dropdownActiveClassDisabled: true,
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            label: 'GitHub',
            to: 'https://github.com/acorn-io/runtime',
          },
          {
            label: 'Users Slack',
            to: 'https://slack.acorn.io',
          },
          {
            label: 'Twitter',
            to: 'https://twitter.com/acornlabs',
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Acorn Labs, Inc`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ['cue','docker'],
      },
      algolia: {
        appId: '7QCEFR54LA',
        apiKey: '0091e059262804a95d3253d28bc90eeb',
        indexName: 'acorn-io',
      }
    })
};

module.exports = config;
