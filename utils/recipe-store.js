const RECIPE_STORAGE_KEY = 'caipu-miniapp-recipes'

export const mealTypeOptions = [
	{ label: '早餐', value: 'breakfast', icon: 'clock-fill', activeColor: '#a06a3f' },
	{ label: '正餐', value: 'main', icon: 'grid-fill', activeColor: '#5b4a3b' }
]

export const statusOptions = [
	{ label: '想吃', value: 'wishlist' },
	{ label: '吃过', value: 'done' }
]

export const mealTypeLabelMap = {
	breakfast: '早餐',
	main: '正餐'
}

export const statusLabelMap = {
	wishlist: '想吃',
	done: '吃过'
}

const defaultRecipes = [
	{
		id: 'r1',
		title: '番茄滑蛋牛肉',
		ingredient: '牛肉',
		link: 'https://www.xiachufang.com/recipe/107000001/',
		image: '',
		mealType: 'main',
		status: 'done',
		note: '下饭',
		parsedContent: {
			ingredients: ['牛里脊 200g', '番茄 2个', '鸡蛋 3个', '蒜末 少许', '生抽、盐、黑胡椒 适量'],
			steps: ['牛肉切片后用少量生抽和黑胡椒抓匀，静置十分钟。', '番茄先炒软出汁，鸡蛋单独滑散备用。', '牛肉快速滑熟，再把番茄和鸡蛋回锅翻匀即可。']
		}
	},
	{
		id: 'r2',
		title: '神仙糖醋排骨',
		ingredient: '排骨',
		link: 'https://www.bilibili.com/video/BV1ab411c7Z9',
		image: '',
		mealType: 'main',
		status: 'wishlist',
		note: '周末吃',
		parsedContent: {
			ingredients: ['小排 500g', '冰糖 适量', '生抽 2勺', '香醋 3勺', '姜片 3片'],
			steps: ['排骨焯水后沥干，先小火煎出表面金黄。', '按糖醋汁比例下锅焖煮，让排骨慢慢收汁入味。', '最后开大火收浓汤汁，表面裹匀亮晶晶的糖醋汁。']
		}
	},
	{
		id: 'r3',
		title: '蒜香虾仁意面',
		ingredient: '虾仁',
		link: 'https://www.xiachufang.com/recipe/104999888/',
		image: '',
		mealType: 'main',
		status: 'wishlist',
		note: '',
		parsedContent: {
			ingredients: ['意面 1把', '虾仁 180g', '蒜末 4瓣', '黄油 1小块', '欧芹碎、盐、黑胡椒 适量'],
			steps: ['意面煮到略带硬芯，留半杯面汤备用。', '黄油炒香蒜末和虾仁，再加入意面和面汤翻匀。', '最后撒欧芹碎和黑胡椒，口味会更像餐厅版。']
		}
	},
	{
		id: 'r4',
		title: '低卡燕麦松饼',
		ingredient: '燕麦',
		link: 'https://www.douyin.com/video/7400000000000000001',
		image: '',
		mealType: 'breakfast',
		status: 'done',
		note: '简单快手',
		parsedContent: {
			ingredients: ['即食燕麦 50g', '香蕉 1根', '鸡蛋 1个', '牛奶 适量', '蜂蜜 少许'],
			steps: ['把燕麦、香蕉、鸡蛋和牛奶打成略稠的面糊。', '平底锅小火分次煎成圆饼，两面上色即可。', '出锅后淋一点蜂蜜或配酸奶，早餐会更完整。']
		}
	},
	{
		id: 'r5',
		title: '周末寿喜锅',
		ingredient: '牛肉',
		link: 'https://www.xiachufang.com/recipe/106123456/',
		image: '',
		mealType: 'main',
		status: 'wishlist',
		note: '',
		parsedContent: {
			ingredients: ['肥牛卷 300g', '娃娃菜 1颗', '豆腐 1盒', '香菇 6朵', '寿喜烧酱汁 1份'],
			steps: ['先把汤底调好，再按耐煮到易熟的顺序摆入食材。', '牛肉最后下锅涮煮，保证口感嫩。', '边煮边吃，汤汁收浓后蘸生鸡蛋液会更日式。']
		}
	},
	{
		id: 'r6',
		title: '牛油果煎蛋吐司',
		ingredient: '牛油果',
		link: 'https://www.bilibili.com/video/BV1xx411x7T8',
		image: '',
		mealType: 'breakfast',
		status: 'wishlist',
		note: '十分钟内',
		parsedContent: {
			ingredients: ['吐司 2片', '牛油果 1个', '鸡蛋 1个', '海盐、黑胡椒 适量', '柠檬汁 少许'],
			steps: ['吐司先烤脆，牛油果压泥后拌一点柠檬汁。', '鸡蛋煎到自己喜欢的熟度，再铺到吐司上。', '最后撒上海盐和黑胡椒，整体会更清爽。']
		}
	}
]

function cloneParsedContent(parsedContent = {}) {
	const ingredients = Array.isArray(parsedContent.ingredients) ? parsedContent.ingredients.filter(Boolean) : []
	const steps = Array.isArray(parsedContent.steps) ? parsedContent.steps.filter(Boolean) : []
	return {
		ingredients,
		steps
	}
}

export function buildFallbackParsedContent(recipe = {}) {
	const mealLabel = mealTypeLabelMap[recipe.mealType] || '这道菜'
	const mainIngredient = (recipe.ingredient || recipe.title || '主食材').trim()

	return {
		ingredients: [
			`${mainIngredient} 1份`,
			`${mealLabel}常用配菜 适量`,
			'基础调味 适量'
		],
		steps: [
			`先从链接里抓出 ${recipe.title || '这道菜'} 的核心做法。`,
			'按自己的口味整理成容易复刻的家常版本。',
			'做完以后回来补充口感、火候和踩坑点。'
		]
	}
}

export function normalizeRecipe(recipe = {}) {
	const normalized = {
		id: recipe.id || `recipe-${Date.now()}`,
		title: (recipe.title || '').trim(),
		ingredient: (recipe.ingredient || '').trim(),
		link: (recipe.link || '').trim(),
		image: recipe.image || '',
		mealType: recipe.mealType || 'breakfast',
		status: recipe.status || 'wishlist',
		note: (recipe.note || '').trim()
	}

	const parsedContent = cloneParsedContent(recipe.parsedContent)
	const hasParsedContent = parsedContent.ingredients.length || parsedContent.steps.length

	return {
		...normalized,
		parsedContent: hasParsedContent ? parsedContent : buildFallbackParsedContent(normalized)
	}
}

export function formatRecipeLink(link = '') {
	const cleaned = link.replace(/^https?:\/\//, '').replace(/^www\./, '').split('?')[0]
	return cleaned.length > 32 ? `${cleaned.slice(0, 29)}...` : cleaned
}

export function getRecipeSecondaryText(recipe = {}) {
	const ingredient = (recipe.ingredient || '').trim()
	const note = (recipe.note || '').trim()
	const link = (recipe.link || '').trim()

	if (ingredient && note) return `${ingredient} · ${note}`
	if (ingredient) return ingredient
	if (note) return note
	if (link) return formatRecipeLink(link)

	const mealLabel = mealTypeLabelMap[recipe.mealType] || '早餐'
	const statusLabel = statusLabelMap[recipe.status] || '想吃'
	return `${mealLabel} · ${statusLabel}`
}

export function saveRecipes(recipes = []) {
	const normalizedRecipes = recipes.map((recipe) => normalizeRecipe(recipe))
	uni.setStorageSync(RECIPE_STORAGE_KEY, normalizedRecipes)
	return normalizedRecipes
}

export function loadRecipes() {
	const storedRecipes = uni.getStorageSync(RECIPE_STORAGE_KEY)
	if (Array.isArray(storedRecipes)) {
		return storedRecipes.map((recipe) => normalizeRecipe(recipe))
	}

	return saveRecipes(defaultRecipes)
}

export function getRecipeById(recipeId) {
	return loadRecipes().find((recipe) => recipe.id === recipeId) || null
}

export function createRecipeFromDraft(draft = {}) {
	return normalizeRecipe({
		id: `recipe-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
		title: draft.title,
		ingredient: draft.ingredient,
		link: draft.link,
		image: draft.image,
		mealType: draft.mealType,
		status: draft.status,
		parsedContent: draft.parsedContent,
		note: draft.note
	})
}

export function updateRecipeById(recipeId, updates = {}) {
	const recipes = loadRecipes()
	const nextRecipes = recipes.map((recipe) => {
		if (recipe.id !== recipeId) return recipe
		return normalizeRecipe({
			...recipe,
			...updates,
			parsedContent: updates.parsedContent || recipe.parsedContent
		})
	})
	return saveRecipes(nextRecipes)
}

export function deleteRecipeById(recipeId) {
	const recipes = loadRecipes().filter((recipe) => recipe.id !== recipeId)
	return saveRecipes(recipes)
}
