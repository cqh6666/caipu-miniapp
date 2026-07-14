import { getCurrentKitchenId } from '../../utils/auth'
import {
	createPlaceFromDraft,
	deletePlaceById,
	getCachedPlaces,
	loadPlaces,
	normalizePlace,
	updatePlaceById,
	updatePlaceStatusById
} from '../../utils/place-store'
import { defineIndexPageModule } from './page-module'

export function createEmptyPlaceDraft(overrides = {}) {
	return {
		name: '',
		type: 'food',
		address: '',
		latitude: 0,
		longitude: 0,
		price: '',
		source: 'manual',
		sourceUrl: '',
		images: [],
		status: 'want',
		tags: [],
		note: '',
		...overrides
	}
}

export function createPlaceDraftFromPlace(place = {}) {
	const normalized = normalizePlace(place)
	return createEmptyPlaceDraft({
		name: normalized.name,
		type: normalized.type,
		address: normalized.address,
		latitude: normalized.latitude,
		longitude: normalized.longitude,
		price: normalized.price,
		source: normalized.source,
		sourceUrl: normalized.sourceUrl,
		images: normalized.imageUrls,
		status: normalized.status,
		tags: normalized.tags,
		note: normalized.note
	})
}

export function normalizePlaceList(places = []) {
	return Array.isArray(places)
		? places.map((item) => normalizePlace(item)).filter((item) => item.id)
		: []
}

export function filterPlaces(places = [], status = 'all', keyword = '') {
	const normalizedKeyword = String(keyword || '').trim().toLowerCase()
	return places.filter((place) => {
		const matchStatus = status === 'all' || place.status === status
		const searchable = [
			place.name,
			place.address,
			place.price,
			place.note,
			...(Array.isArray(place.tags) ? place.tags : [])
		].join(' ').toLowerCase()
		return matchStatus && (!normalizedKeyword || searchable.includes(normalizedKeyword))
	})
}

export function parsePlaceListInput(value = '', maxItems = Infinity) {
	return String(value || '')
		.split(/[，,\n]/)
		.map((item) => item.trim())
		.filter(Boolean)
		.slice(0, maxItems)
}

export function countPlacesByStatus(places = [], status = 'all') {
	return status === 'all'
		? places.length
		: places.filter((place) => place.status === status).length
}


export const placeLibraryMethods = {
	handlePlaceOpen(placeId) {
		const targetPlaceId = String(placeId || '').trim()
		if (!targetPlaceId) return
		this.selectedPlaceId = targetPlaceId
		this.showPlaceDetailSheet = true
	},
	applyPlaces(places = []) {
		this.places = Array.isArray(places) ? places.filter((item) => item?.id) : []
	},
	async refreshPlaces(options = {}) {
		const { silent = true } = options
		this.applyPlaces(getCachedPlaces())

		try {
			this.isLoadingPlaces = true
			const places = await loadPlaces({ forceRefresh: true })
			this.placeSyncErrorMessage = ''
			this.applyPlaces(places)
			return places
		} catch (error) {
			this.placeSyncErrorMessage = error?.message || '同步打卡点失败'
			this.applyPlaces(getCachedPlaces())
			if (!silent) {
				uni.showToast({
					title: error?.message || '同步打卡点失败',
					icon: 'none'
				})
			}
			return this.places
		} finally {
			this.isLoadingPlaces = false
		}
	},
	openPlaceCreateSheet() {
		if (!getCurrentKitchenId()) {
			uni.showToast({
				title: '请先完成空间同步',
				icon: 'none'
			})
			return
		}
		// 打开智能识别面板
		this.showAddLinkPreviewPanel = true
	},
	openPlaceEditSheet(placeId = '') {
		const targetPlaceId = String(placeId || this.selectedPlaceId || '').trim()
		const place = this.places.find((item) => item.id === targetPlaceId)
		if (!place) return
		this.placeEditMode = 'edit'
		this.editingPlaceId = targetPlaceId
		this.placeDraft = createPlaceDraftFromPlace(place)
		this.showPlaceDetailSheet = false
		this.showPlaceEditSheet = true
	},
	closePlaceEditSheet() {
		if (this.isSubmittingPlace) return
		this.showPlaceEditSheet = false
		this.placeEditMode = 'create'
		this.editingPlaceId = ''
		this.placeDraft = createEmptyPlaceDraft()
	},
	closePlaceDetailSheet() {
		if (this.isSubmittingPlace) return
		this.showPlaceDetailSheet = false
	},
	closePlaceExperienceSheet() {
		if (this.isSubmittingPlace) return
		this.showPlaceExperienceSheet = false
		this.pendingPlaceStatusChangeId = ''
	},
	async handleExperienceSkip() {
		// 跳过体验采集，直接切换状态
		const targetPlaceId = this.pendingPlaceStatusChangeId
		this.closePlaceExperienceSheet()

		if (!targetPlaceId) return

		this.isSubmittingPlace = true
		try {
			const updated = await updatePlaceStatusById(targetPlaceId, 'visited')
			this.applyPlaces(getCachedPlaces())
			this.selectedPlaceId = updated?.id || targetPlaceId
			uni.showToast({
				title: '已标记去过',
				icon: 'none'
			})
		} catch (error) {
			uni.showToast({
				title: error?.message || '更新打卡状态失败',
				icon: 'none'
			})
		} finally {
			this.isSubmittingPlace = false
		}
	},
	async handleExperienceSubmit(experienceData) {
		// 保存体验数据并切换状态
		const targetPlaceId = this.pendingPlaceStatusChangeId
		this.closePlaceExperienceSheet()

		if (!targetPlaceId) return

		this.isSubmittingPlace = true
		try {
			const updated = await updatePlaceStatusById(targetPlaceId, 'visited', experienceData)
			this.applyPlaces(getCachedPlaces())
			this.selectedPlaceId = updated?.id || targetPlaceId
			uni.showToast({
				title: '打卡完成！',
				icon: 'success'
			})
		} catch (error) {
			uni.showToast({
				title: error?.message || '保存体验失败',
				icon: 'none'
			})
		} finally {
			this.isSubmittingPlace = false
		}
	},
	handlePlaceDraftNameInput(event) {
		this.placeDraft.name = String(event?.detail?.value || '')
	},
	handlePlaceDraftStatusSelect(value) {
		if (!this.placeStatusOptions.some((item) => item.value === value)) return
		this.placeDraft.status = value
	},
	handlePlaceDraftTypeSelect(value) {
		if (!this.placeTypeOptions.some((item) => item.value === value)) return
		this.placeDraft.type = value
	},
	handlePlaceDraftAddressInput(event) {
		this.placeDraft.address = String(event?.detail?.value || '')
	},
	handlePlaceDraftPriceInput(event) {
		this.placeDraft.price = String(event?.detail?.value || '')
	},
	handlePlaceDraftTagsInput(event) {
		this.placeDraft.tags = parsePlaceListInput(event?.detail?.value || '')
	},
	handlePlaceDraftSourceSelect(value) {
		if (!this.placeSourceOptions.some((item) => item.value === value)) return
		this.placeDraft.source = value
	},
	handlePlaceDraftSourceUrlInput(event) {
		this.placeDraft.sourceUrl = String(event?.detail?.value || '')
	},
	handlePlaceDraftNoteInput(event) {
		this.placeDraft.note = String(event?.detail?.value || '')
	},
	handlePlaceDraftPhoneInput(event) {
		this.placeDraft.phone = String(event?.detail?.value || '')
	},
	handlePlaceDraftRatingInput(rating) {
		this.placeDraft.revisitRating = Number(rating) || 0
	},
	handlePlaceDraftRecommendedItemsInput(event) {
		this.placeDraft.recommendedItems = parsePlaceListInput(event?.detail?.value || '', 12)
	},
	handlePlaceDraftDiningTipsInput(event) {
		this.placeDraft.diningTips = String(event?.detail?.value || '')
	},
	handlePlaceDraftScenesInput(event) {
		this.placeDraft.scenes = parsePlaceListInput(event?.detail?.value || '', 8)
	},
	handlePlaceDraftBestTimeInput(event) {
		this.placeDraft.bestTime = String(event?.detail?.value || '')
	},
	handlePlaceDraftDurationInput(event) {
		this.placeDraft.duration = String(event?.detail?.value || '')
	},
	handlePlaceDraftCompanionTagsInput(event) {
		this.placeDraft.companionTags = parsePlaceListInput(event?.detail?.value || '', 6)
	},
	handlePlaceDraftParkingNoteInput(event) {
		this.placeDraft.parkingNote = String(event?.detail?.value || '')
	},
	choosePlaceLocation() {
		if (typeof uni.chooseLocation !== 'function') {
			uni.showToast({
				title: '当前环境不支持地图选点',
				icon: 'none'
			})
			return
		}
		uni.chooseLocation({
			success: (result) => {
				const nextName = String(result?.name || '').trim()
				const nextAddress = String(result?.address || '').trim()
				if (nextName && !String(this.placeDraft.name || '').trim()) {
					this.placeDraft.name = nextName
				}
				this.placeDraft.address = nextAddress || nextName || this.placeDraft.address
				this.placeDraft.latitude = Number(result?.latitude) || 0
				this.placeDraft.longitude = Number(result?.longitude) || 0
			},
			fail: (error) => {
				const message = String(error?.errMsg || '')
				if (message.includes('cancel')) return
				uni.showToast({
					title: '地图选点失败',
					icon: 'none'
				})
			}
		})
	},
	choosePlaceImages() {
		const images = Array.isArray(this.placeDraft.images) ? this.placeDraft.images : []
		const remaining = Math.max(this.maxPlaceImages - images.length, 0)
		if (!remaining) {
			uni.showToast({
				title: `最多上传 ${this.maxPlaceImages} 张`,
				icon: 'none'
			})
			return
		}

		uni.chooseImage({
			count: remaining,
			sizeType: ['compressed'],
			sourceType: ['album', 'camera'],
			success: ({ tempFilePaths }) => {
				if (!tempFilePaths || !tempFilePaths.length) return
				const nextImages = [...images]
				tempFilePaths.forEach((path) => {
					if (path && !nextImages.includes(path) && nextImages.length < this.maxPlaceImages) {
						nextImages.push(path)
					}
				})
				this.placeDraft.images = nextImages
			}
		})
	},
	removePlaceDraftImage(index) {
		if (typeof index !== 'number') return
		this.placeDraft.images = (Array.isArray(this.placeDraft.images) ? this.placeDraft.images : [])
			.filter((_, currentIndex) => currentIndex !== index)
	},
	previewPlaceDraftImages(index = 0) {
		const urls = Array.isArray(this.placeDraft.images) ? this.placeDraft.images.filter(Boolean) : []
		if (!urls.length) return
		uni.previewImage({
			current: urls[index] || urls[0],
			urls
		})
	},
	previewSelectedPlaceImages(index = 0) {
		const urls = Array.isArray(this.selectedPlace?.imageUrls) ? this.selectedPlace.imageUrls.filter(Boolean) : []
		if (!urls.length) return
		uni.previewImage({
			current: urls[index] || urls[0],
			urls
		})
	},
	async submitPlaceDraft() {
		if (!this.canSubmitPlaceDraft || this.isSubmittingPlace) return
		this.isSubmittingPlace = true
		try {
			const saved = this.placeEditMode === 'edit' && this.editingPlaceId
				? await updatePlaceById(this.editingPlaceId, this.placeDraft)
				: await createPlaceFromDraft(this.placeDraft)
			this.applyPlaces(getCachedPlaces())
			this.selectedPlaceId = saved?.id || this.selectedPlaceId
			this.showPlaceEditSheet = false
			this.showPlaceDetailSheet = !!saved?.id
			this.placeEditMode = 'create'
			this.editingPlaceId = ''
			this.placeDraft = createEmptyPlaceDraft()
			uni.showToast({
				title: '打卡点已保存',
				icon: 'none'
			})
		} catch (error) {
			uni.showToast({
				title: error?.message || '保存打卡点失败',
				icon: 'none'
			})
		} finally {
			this.isSubmittingPlace = false
		}
	},
	openPlaceLocation(placeId = '') {
		const targetPlaceId = String(placeId || this.selectedPlaceId || '').trim()
		const place = this.places.find((item) => item.id === targetPlaceId)
		if (!place) return
		const latitude = Number(place.latitude) || 0
		const longitude = Number(place.longitude) || 0
		if (!latitude || !longitude || typeof uni.openLocation !== 'function') {
			uni.showToast({
				title: '还没有可打开的位置',
				icon: 'none'
			})
			return
		}
		uni.openLocation({
			latitude,
			longitude,
			name: place.name || '打卡点',
			address: place.address || '',
			fail: () => {
				uni.showToast({
					title: '打开地图失败',
					icon: 'none'
				})
			}
		})
	},
	openPlaceSourceURL(placeId = '') {
		const targetPlaceId = String(placeId || this.selectedPlaceId || '').trim()
		const place = this.places.find((item) => item.id === targetPlaceId)
		const sourceUrl = String(place?.sourceUrl || '').trim()
		if (!sourceUrl) {
			uni.showToast({
				title: '暂无来源链接',
				icon: 'none'
			})
			return
		}
		uni.setClipboardData({
			data: sourceUrl,
			success: () => {
				uni.showToast({
					title: '来源链接已复制',
					icon: 'none'
				})
			},
			fail: () => {
				uni.showToast({
					title: '复制链接失败',
					icon: 'none'
				})
			}
		})
	},
	async togglePlaceStatus(placeId = '') {
		const targetPlaceId = String(placeId || this.selectedPlaceId || '').trim()
		const place = this.places.find((item) => item.id === targetPlaceId)
		if (!place || this.isSubmittingPlace) return

		const nextStatus = place.status === 'visited' ? 'want' : 'visited'

		// 如果切换到"去过"，先弹出体验采集弹窗
		if (nextStatus === 'visited') {
			this.pendingPlaceStatusChangeId = targetPlaceId
			this.showPlaceExperienceSheet = true
			return
		}

		// 切回"想去"直接执行
		this.isSubmittingPlace = true
		try {
			const updated = await updatePlaceStatusById(targetPlaceId, nextStatus)
			this.applyPlaces(getCachedPlaces())
			this.selectedPlaceId = updated?.id || targetPlaceId
			uni.showToast({
				title: '已改回想去',
				icon: 'none'
			})
		} catch (error) {
			uni.showToast({
				title: error?.message || '更新打卡状态失败',
				icon: 'none'
			})
		} finally {
			this.isSubmittingPlace = false
		}
	},
	confirmDeletePlace(placeId = '') {
		const targetPlaceId = String(placeId || this.selectedPlaceId || '').trim()
		const place = this.places.find((item) => item.id === targetPlaceId)
		if (!place || this.isSubmittingPlace) return
		uni.showModal({
			title: '删除打卡点',
			content: `确认删除「${place.name || '这个打卡点'}」吗？`,
			confirmText: '删除',
			confirmColor: '#a95549',
			success: async ({ confirm }) => {
				if (!confirm) return
				await this.deletePlace(targetPlaceId)
			}
		})
	},
	async deletePlace(placeId = '') {
		const targetPlaceId = String(placeId || '').trim()
		if (!targetPlaceId || this.isSubmittingPlace) return
		this.isSubmittingPlace = true
		try {
			await deletePlaceById(targetPlaceId)
			this.applyPlaces(getCachedPlaces())
			if (this.selectedPlaceId === targetPlaceId) {
				this.selectedPlaceId = ''
				this.showPlaceDetailSheet = false
			}
			uni.showToast({
				title: '打卡点已删除',
				icon: 'none'
			})
		} catch (error) {
			uni.showToast({
				title: error?.message || '删除打卡点失败',
				icon: 'none'
			})
		} finally {
			this.isSubmittingPlace = false
		}
	},
	// 智能识别相关方法
}

export const placeLibraryComputed = {
		trimmedPlaceSearchKeyword() {
			return String(this.placeSearchKeyword || '').trim()
		},
		filteredPlaces() {
			return filterPlaces(this.places, this.activePlaceStatus, this.trimmedPlaceSearchKeyword)
		},
		selectedPlace() {
			const targetPlaceId = String(this.selectedPlaceId || '').trim()
			return this.places.find((place) => place.id === targetPlaceId) || {}
		},
		pendingPlaceType() {
			const targetPlaceId = String(this.pendingPlaceStatusChangeId || '').trim()
			const place = this.places.find((p) => p.id === targetPlaceId)
			return place?.type || 'food'
		},
		canSubmitPlaceDraft() {
			return !!String(this.placeDraft.name || '').trim()
		}
}

export const placeLibraryModule = defineIndexPageModule({
	name: 'place-library',
	requires: ['places', 'activePlaceStatus', 'placeSearchKeyword', 'placeDraft'],
	methods: placeLibraryMethods,
	computed: placeLibraryComputed,
	lifecycle: {
		onKitchenChange({ nextKitchenId = 0 } = {}) {
			this.selectedPlaceId = ''
			this.showPlaceDetailSheet = false
			this.showPlaceEditSheet = false
			this.applyPlaces(nextKitchenId ? getCachedPlaces(nextKitchenId) : [])
		}
	}
})
