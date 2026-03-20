import { isServerAssetPath, resolveAssetURL } from './app-config'
import { uploadFile } from './http'

function isRemoteImage(url = '') {
	return /^https?:\/\//.test((url || '').trim())
}

export async function ensureUploadedImage(imagePath = '') {
	const source = (imagePath || '').trim()
	if (!source) return ''
	if (isRemoteImage(source) || isServerAssetPath(source)) return resolveAssetURL(source)

	const payload = await uploadFile({
		url: '/caipu-api/uploads/images',
		filePath: source,
		name: 'file'
	})

	return resolveAssetURL(payload?.url || '')
}

export async function ensureUploadedImages(imagePaths = []) {
	const sources = Array.isArray(imagePaths) ? imagePaths : [imagePaths]
	const uploaded = []

	for (const item of sources) {
		const url = await ensureUploadedImage(item)
		if (url) {
			uploaded.push(url)
		}
	}

	return uploaded
}
