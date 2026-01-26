declare module 'virtual:starlight/pagefind-config' {
	export const pagefindUserConfig: Partial<
		Extract<import('./types').StarlightConfig['pagefind'], object>
	>;
}
