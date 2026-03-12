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
