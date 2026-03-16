import { uploadFile } from './http'

function isRemoteImage(url = '') {
	return /^https?:\/\//.test((url || '').trim())
}

export async function ensureUploadedImage(imagePath = '') {
	const source = (imagePath || '').trim()
	if (!source) return ''
	if (isRemoteImage(source)) return source

	const payload = await uploadFile({
		url: '/api/uploads/images',
		filePath: source,
		name: 'file'
	})

	return payload?.url || ''
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
