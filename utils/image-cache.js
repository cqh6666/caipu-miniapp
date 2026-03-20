const IMAGE_CACHE_STORAGE_KEY = 'caipu-miniapp-image-cache:v1'
const DEFAULT_IMAGE_CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000
const MAX_IMAGE_CACHE_RECORDS = 80

const pendingImageTasks = new Map()

function normalizeURL(url = '') {
	return String(url || '').trim()
}

function normalizeVersion(version = '') {
	return String(version || '').trim()
}

function getFileSystemManager() {
	if (typeof uni !== 'undefined' && typeof uni.getFileSystemManager === 'function') {
		return uni.getFileSystemManager()
	}
	if (typeof wx !== 'undefined' && typeof wx.getFileSystemManager === 'function') {
		return wx.getFileSystemManager()
	}
	return null
}

function canUsePersistentImageCache() {
	return (
		typeof uni !== 'undefined' &&
		typeof uni.getStorageSync === 'function' &&
		typeof uni.setStorageSync === 'function' &&
		typeof uni.downloadFile === 'function' &&
		!!getFileSystemManager()
	)
}

function getStoredImageCacheRecords() {
	if (!canUsePersistentImageCache()) return {}
	const stored = uni.getStorageSync(IMAGE_CACHE_STORAGE_KEY)
	if (!stored || typeof stored !== 'object' || Array.isArray(stored)) return {}
	return stored
}

function saveImageCacheRecords(records = {}) {
	if (!canUsePersistentImageCache()) return
	uni.setStorageSync(IMAGE_CACHE_STORAGE_KEY, records)
}

function getCacheTTL(ttlMs) {
	return Number.isFinite(ttlMs) && ttlMs > 0 ? ttlMs : DEFAULT_IMAGE_CACHE_TTL_MS
}

function isRecordExpired(record = {}, ttlMs = DEFAULT_IMAGE_CACHE_TTL_MS) {
	const cachedAt = Number(record.cachedAt) || 0
	if (!cachedAt) return false
	return Date.now() - cachedAt > ttlMs
}

function accessLocalFile(filePath = '') {
	const fs = getFileSystemManager()
	const path = String(filePath || '').trim()
	if (!fs || !path) return Promise.resolve(false)

	return new Promise((resolve) => {
		fs.access({
			path,
			success: () => resolve(true),
			fail: () => resolve(false)
		})
	})
}

function removeSavedFile(filePath = '') {
	const fs = getFileSystemManager()
	const path = String(filePath || '').trim()
	if (!fs || !path) return Promise.resolve(false)

	return new Promise((resolve) => {
		fs.removeSavedFile({
			filePath: path,
			success: () => resolve(true),
			fail: () => resolve(false)
		})
	})
}

function downloadRemoteFile(url = '') {
	const targetURL = normalizeURL(url)
	if (!targetURL || !canUsePersistentImageCache()) {
		return Promise.resolve(null)
	}

	return new Promise((resolve, reject) => {
		uni.downloadFile({
			url: targetURL,
			success: (result) => {
				const statusCode = Number(result?.statusCode) || 0
				if (statusCode < 200 || statusCode >= 300) {
					reject(new Error(`download file failed: ${statusCode || 'unknown'}`))
					return
				}
				resolve(result)
			},
			fail: reject
		})
	})
}

function saveTempFile(tempFilePath = '') {
	const fs = getFileSystemManager()
	const path = String(tempFilePath || '').trim()
	if (!fs || !path) return Promise.resolve('')

	return new Promise((resolve, reject) => {
		fs.saveFile({
			tempFilePath: path,
			success: (result) => resolve(String(result?.savedFilePath || '').trim()),
			fail: reject
		})
	})
}

async function pruneImageCacheRecords(records = {}, options = {}) {
	const ttlMs = getCacheTTL(options.ttlMs)
	const preserveKeys = new Set(Array.isArray(options.preserveKeys) ? options.preserveKeys : [])
	const nextRecords = { ...records }
	const removable = []

	Object.keys(nextRecords).forEach((key) => {
		if (preserveKeys.has(key)) return
		if (isRecordExpired(nextRecords[key], ttlMs)) {
			removable.push(key)
		}
	})

	removable.forEach((key) => {
		delete nextRecords[key]
	})

	const activeKeys = Object.keys(nextRecords)
	const overflowCount = activeKeys.length - MAX_IMAGE_CACHE_RECORDS
	if (overflowCount > 0) {
		const overflowKeys = activeKeys
			.filter((key) => !preserveKeys.has(key))
			.sort((left, right) => {
				const leftRecord = nextRecords[left] || {}
				const rightRecord = nextRecords[right] || {}
				const leftTime = Number(leftRecord.lastAccessAt) || Number(leftRecord.cachedAt) || 0
				const rightTime = Number(rightRecord.lastAccessAt) || Number(rightRecord.cachedAt) || 0
				return leftTime - rightTime
			})
			.slice(0, overflowCount)

		overflowKeys.forEach((key) => {
			removable.push(key)
			delete nextRecords[key]
		})
	}

	saveImageCacheRecords(nextRecords)

	await Promise.all(
		removable.map((key) => {
			const filePath = String(records[key]?.filePath || '').trim()
			return removeSavedFile(filePath)
		})
	)
}

export function buildImageCacheKey(url = '', version = '') {
	return `${normalizeURL(url)}::${normalizeVersion(version)}`
}

export async function getCachedImagePath(url = '', version = '', options = {}) {
	if (!canUsePersistentImageCache()) return ''

	const cacheKey = buildImageCacheKey(url, version)
	if (!cacheKey || cacheKey === '::') return ''

	const ttlMs = getCacheTTL(options.ttlMs)
	const records = getStoredImageCacheRecords()
	const record = records[cacheKey]
	if (!record?.filePath) return ''

	if (isRecordExpired(record, ttlMs)) {
		delete records[cacheKey]
		saveImageCacheRecords(records)
		await removeSavedFile(record.filePath)
		return ''
	}

	const exists = await accessLocalFile(record.filePath)
	if (!exists) {
		delete records[cacheKey]
		saveImageCacheRecords(records)
		return ''
	}

	if (options.touch !== false) {
		records[cacheKey] = {
			...record,
			lastAccessAt: Date.now()
		}
		saveImageCacheRecords(records)
	}

	return String(record.filePath || '').trim()
}

export async function ensureCachedImage(url = '', version = '', options = {}) {
	const targetURL = normalizeURL(url)
	if (!targetURL || !canUsePersistentImageCache()) return ''

	const cacheKey = buildImageCacheKey(targetURL, version)
	const cachedPath = await getCachedImagePath(targetURL, version, options)
	if (cachedPath) return cachedPath

	if (pendingImageTasks.has(cacheKey)) {
		return pendingImageTasks.get(cacheKey)
	}

	const task = (async () => {
		try {
			const downloadResult = await downloadRemoteFile(targetURL)
			const tempFilePath = String(downloadResult?.tempFilePath || '').trim()
			if (!tempFilePath) return ''

			const savedFilePath = await saveTempFile(tempFilePath)
			if (!savedFilePath) return ''

			const nextRecords = getStoredImageCacheRecords()
			const orphanFilePaths = []

			Object.keys(nextRecords).forEach((key) => {
				const record = nextRecords[key]
				if (!record || normalizeURL(record.url) !== targetURL || key === cacheKey) return
				if (record.filePath) {
					orphanFilePaths.push(record.filePath)
				}
				delete nextRecords[key]
			})

			nextRecords[cacheKey] = {
				url: targetURL,
				version: normalizeVersion(version),
				filePath: savedFilePath,
				cachedAt: Date.now(),
				lastAccessAt: Date.now()
			}

			saveImageCacheRecords(nextRecords)
			await Promise.all(orphanFilePaths.map((filePath) => removeSavedFile(filePath)))
			await pruneImageCacheRecords(nextRecords, {
				ttlMs: options.ttlMs,
				preserveKeys: [cacheKey]
			})

			return savedFilePath
		} catch (error) {
			return ''
		} finally {
			pendingImageTasks.delete(cacheKey)
		}
	})()

	pendingImageTasks.set(cacheKey, task)
	return task
}

export async function warmImageCache(entries = [], options = {}) {
	if (!Array.isArray(entries) || !entries.length || !canUsePersistentImageCache()) return

	const concurrency = Math.max(1, Math.min(Number(options.concurrency) || 2, 4))
	const deduped = []
	const seen = new Set()

	entries.forEach((entry) => {
		const url = normalizeURL(entry?.url)
		if (!url) return
		const cacheKey = buildImageCacheKey(url, entry?.version)
		if (seen.has(cacheKey)) return
		seen.add(cacheKey)
		deduped.push({
			url,
			version: normalizeVersion(entry?.version),
			cacheKey
		})
	})

	let cursor = 0
	const worker = async () => {
		while (cursor < deduped.length) {
			const current = deduped[cursor]
			cursor += 1
			const localPath = await ensureCachedImage(current.url, current.version, options)
			if (localPath && typeof options.onResolved === 'function') {
				options.onResolved({
					...current,
					localPath
				})
			}
		}
	}

	const workers = Array.from({ length: Math.min(concurrency, deduped.length) }, () => worker())
	await Promise.all(workers)
}

export async function clearImageCache() {
	if (!canUsePersistentImageCache()) {
		return {
			removedCount: 0
		}
	}

	const records = getStoredImageCacheRecords()
	const filePaths = Object.values(records)
		.map((record) => String(record?.filePath || '').trim())
		.filter(Boolean)

	saveImageCacheRecords({})
	await Promise.all(filePaths.map((filePath) => removeSavedFile(filePath)))

	return {
		removedCount: filePaths.length
	}
}
