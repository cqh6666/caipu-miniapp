export function defineIndexPageModule(definition = {}) {
	const name = String(definition.name || '').trim()
	if (!name) throw new Error('index page module name is required')
	return Object.freeze({
		name,
		requires: [...new Set(definition.requires || [])],
		methods: definition.methods || {},
		computed: definition.computed || {},
		lifecycle: definition.lifecycle || {}
	})
}

export function installIndexPageModules(modules = []) {
	const methods = {}
	const computed = {}
	const owners = new Map()

	modules.forEach((module) => {
		Object.entries(module.methods).forEach(([key, value]) => {
			if (owners.has(`method:${key}`)) {
				throw new Error(`duplicate index page method "${key}" in ${owners.get(`method:${key}`)} and ${module.name}`)
			}
			owners.set(`method:${key}`, module.name)
			methods[key] = value
		})
		Object.entries(module.computed).forEach(([key, value]) => {
			if (owners.has(`computed:${key}`)) {
				throw new Error(`duplicate index page computed "${key}" in ${owners.get(`computed:${key}`)} and ${module.name}`)
			}
			owners.set(`computed:${key}`, module.name)
			computed[key] = value
		})
	})

	return { methods, computed }
}

export function validateIndexPageModuleContext(context, modules = []) {
	return modules.flatMap((module) => module.requires
		.filter((key) => !(key in context))
		.map((key) => `${module.name}:${key}`))
}

export function runIndexPageModuleLifecycle(context, modules = [], hook, payload) {
	const hooks = hook === 'dispose' ? ['deactivate', 'dispose'] : [hook]
	hooks.forEach((hookName) => {
		modules.forEach((module) => {
			const handler = module.lifecycle?.[hookName]
			if (typeof handler === 'function') handler.call(context, payload)
		})
	})
}
