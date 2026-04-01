export const statusMap = {
	all: { label: '全部', icon: 'list-dot' },
	wishlist: { label: '想吃', icon: 'heart-fill' },
	done: { label: '吃过', icon: 'checkmark-circle-fill' }
}

export const createEmptyDraft = (overrides = {}) => ({
	title: '',
	link: '',
	images: [],
	mealType: 'breakfast',
	status: 'wishlist',
	note: '',
	...overrides
})

export const MAX_RECENT_SEARCHES = 6

export const searchSuggestionKeywordsByMeal = {
	breakfast: ['鸡蛋', '面包', '粥', '快手'],
	main: ['下饭', '牛肉', '鸡翅', '汤']
}
