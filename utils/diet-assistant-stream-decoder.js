function toUint8Array(buffer) {
	if (typeof buffer === 'string') return buffer
	const source = buffer?.data || buffer
	if (!source) return ''
	if (source instanceof Uint8Array) return source
	return new Uint8Array(source)
}

function codePointToString(codePoint) {
	if (typeof String.fromCodePoint === 'function') return String.fromCodePoint(codePoint)
	const offset = codePoint - 0x10000
	return String.fromCharCode((offset >> 10) + 0xd800, (offset % 0x400) + 0xdc00)
}

export function createManualUTF8Decoder() {
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
			return bytes ? decodeBytes(bytes) : ''
		},
		flush() {
			if (!pending.length) return ''
			pending = []
			return '\ufffd'
		}
	}
}

export function createChunkDecoder(options = {}) {
	const TextDecoderClass = Object.prototype.hasOwnProperty.call(options, 'TextDecoderClass')
		? options.TextDecoderClass
		: typeof TextDecoder !== 'undefined' ? TextDecoder : null
	if (!TextDecoderClass) return createManualUTF8Decoder()

	const decoder = new TextDecoderClass('utf-8')
	return {
		decode(buffer) {
			if (typeof buffer === 'string') return buffer
			const bytes = toUint8Array(buffer)
			return bytes ? decoder.decode(bytes, { stream: true }) : ''
		},
		flush() {
			return decoder.decode()
		}
	}
}
