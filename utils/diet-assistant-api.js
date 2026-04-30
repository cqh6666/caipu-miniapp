import { resolveAPIURL } from './app-config'
import { getAccessToken, getSessionState } from './session-storage'

function normalizeMessages(messages = []) {
	return (Array.isArray(messages) ? messages : [])
		.map((message) => ({
			role: String(message?.role || '').trim(),
			content: String(message?.content || '').trim()
		}))
		.filter((message) => message.role && message.content)
}

function toUint8Array(buffer) {
	if (typeof buffer === 'string') return buffer
	const source = buffer?.data || buffer
	if (!source) return ''
	if (source instanceof Uint8Array) return source
	return new Uint8Array(source)
}

function codePointToString(codePoint) {
	if (typeof String.fromCodePoint === 'function') {
		return String.fromCodePoint(codePoint)
	}
	codePoint -= 0x10000
	return String.fromCharCode((codePoint >> 10) + 0xd800, (codePoint % 0x400) + 0xdc00)
}

function createManualUTF8Decoder() {
	let pending = []

	function decodeBytes(inputBytes) {
		let bytes = inputBytes
		if (pending.length) {
			bytes = new Uint8Array(pending.length + inputBytes.length)
			bytes.set(pending, 0)
			bytes.set(inputBytes, pending.length)
		}
		pending = []
		let result = ''
		let index = 0

		while (index < bytes.length) {
			const first = bytes[index]

			if (first < 0x80) {
				result += String.fromCharCode(first)
				index += 1
				continue
			}

			let needed = 0
			let codePoint = 0
			if (first >= 0xc2 && first <= 0xdf) {
				needed = 2
				codePoint = first & 0x1f
			} else if (first >= 0xe0 && first <= 0xef) {
				needed = 3
				codePoint = first & 0x0f
			} else if (first >= 0xf0 && first <= 0xf4) {
				needed = 4
				codePoint = first & 0x07
			} else {
				result += '\ufffd'
				index += 1
				continue
			}

			if (index + needed > bytes.length) {
				pending = Array.from(bytes.slice(index))
				break
			}

			let valid = true
			for (let offset = 1; offset < needed; offset += 1) {
				const next = bytes[index + offset]
				if ((next & 0xc0) !== 0x80) {
					valid = false
					break
				}
				codePoint = (codePoint << 6) | (next & 0x3f)
			}

			if (!valid) {
				result += '\ufffd'
				index += 1
				continue
			}

			const isOverlong = (needed === 3 && codePoint < 0x800) || (needed === 4 && codePoint < 0x10000)
			const isSurrogate = codePoint >= 0xd800 && codePoint <= 0xdfff
			if (isOverlong || isSurrogate || codePoint > 0x10ffff) {
				result += '\ufffd'
				index += needed
				continue
			}

			result += codePointToString(codePoint)
			index += needed
		}

		return result
	}

	return {
		decode(buffer) {
			if (typeof buffer === 'string') return buffer
			const bytes = toUint8Array(buffer)
			if (!bytes) return ''
			return decodeBytes(bytes)
		},
		flush() {
			if (!pending.length) return ''
			pending = []
			return '\ufffd'
		}
	}
}

function createChunkDecoder() {
	if (typeof TextDecoder !== 'undefined') {
		const decoder = new TextDecoder('utf-8')
		return {
			decode(buffer) {
				if (typeof buffer === 'string') return buffer
				const bytes = toUint8Array(buffer)
				if (!bytes) return ''
				return decoder.decode(bytes, { stream: true })
			},
			flush() {
				return decoder.decode()
			}
		}
	}
	return createManualUTF8Decoder()
}

function createStreamParser(callbacks = {}) {
	let buffer = ''

	function handleFrame(frame = '') {
		const dataLines = frame
			.split('\n')
			.map((line) => line.trim())
			.filter((line) => line.startsWith('data:'))
			.map((line) => line.slice(5).trim())
			.filter(Boolean)

		dataLines.forEach((line) => {
			if (line === '[DONE]') {
				callbacks.onDone?.()
				return
			}

			let event
			try {
				event = JSON.parse(line)
			} catch (error) {
				return
			}

			if (event?.type === 'delta') {
				callbacks.onDelta?.(String(event.delta || ''))
				return
			}
			if (event?.type === 'error') {
				callbacks.onError?.(new Error(event.message || '饮食管家暂时不可用'))
				return
			}
			if (event?.type === 'done') {
				callbacks.onDone?.()
			}
		})
	}

	return {
		push(text = '') {
			buffer += String(text || '').replace(/\r\n/g, '\n')
			let splitIndex = buffer.indexOf('\n\n')
			while (splitIndex >= 0) {
				const frame = buffer.slice(0, splitIndex)
				buffer = buffer.slice(splitIndex + 2)
				handleFrame(frame)
				splitIndex = buffer.indexOf('\n\n')
			}
		},
		flush() {
			const remaining = buffer.trim()
			buffer = ''
			if (remaining) {
				handleFrame(remaining)
			}
		}
	}
}

export function streamDietAssistantChat(messages = [], callbacks = {}) {
	const token = getAccessToken()
	const kitchenId = Number(getSessionState()?.currentKitchenId) || 0
	const decoder = createChunkDecoder()
	let receivedChunk = false
	let requestTask = null
	let streamError = null
	let settled = false
	let resolveFinished
	let rejectFinished

	function settle(handler, value) {
		if (settled) return
		settled = true
		if (typeof handler === 'function') {
			handler(value)
		}
	}

	function resolveOnce() {
		settle(resolveFinished, undefined)
	}

	function rejectOnce(error) {
		settle(rejectFinished, error)
	}

	const parser = createStreamParser({
		...callbacks,
		onError: (error) => {
			streamError = error
			callbacks.onError?.(error)
			rejectOnce(error)
		},
		onDone: () => {
			callbacks.onDone?.()
			resolveOnce()
		}
	})

	const finished = new Promise((resolve, reject) => {
		resolveFinished = resolve
		rejectFinished = reject
		requestTask = uni.request({
			url: resolveAPIURL('/caipu-api/diet-assistant/chat/stream'),
			method: 'POST',
			data: {
				messages: normalizeMessages(messages),
				kitchenId
			},
			header: {
				Authorization: token ? `Bearer ${token}` : '',
				'Content-Type': 'application/json',
				Accept: 'text/event-stream'
			},
			timeout: 90000,
			enableChunked: true,
			responseType: 'arraybuffer',
			success: (response) => {
				const statusCode = Number(response?.statusCode) || 0
				if (statusCode < 200 || statusCode >= 300) {
					rejectOnce(new Error(`饮食管家请求失败 (${statusCode || 'unknown'})`))
					return
				}

				if (!receivedChunk && response?.data) {
					parser.push(decoder.decode(response.data))
				}
				parser.push(decoder.flush())
				parser.flush()
				if (streamError) {
					rejectOnce(streamError)
					return
				}
				resolveOnce()
			},
			fail: (error) => {
				if (settled) return
				rejectOnce(new Error(error?.errMsg || '饮食管家网络请求失败'))
			}
		})

		if (requestTask && typeof requestTask.onChunkReceived === 'function') {
			requestTask.onChunkReceived((chunk) => {
				receivedChunk = true
				parser.push(decoder.decode(chunk?.data || chunk))
			})
		}
	})

	return {
		finished,
		abort() {
			if (requestTask && typeof requestTask.abort === 'function') {
				requestTask.abort()
			}
		},
		setStreamError(error) {
			streamError = error
		}
	}
}
