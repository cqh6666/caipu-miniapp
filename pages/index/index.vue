<template>
	<view class="page">
		<up-notice-bar :text="noticeText" mode="closable"></up-notice-bar>

		<view class="hero">
			<text class="hero__eyebrow">uview-plus 已接入</text>
			<text class="hero__title">开始搭你的菜谱小程序界面</text>
			<text class="hero__desc">
				下面的输入框、按钮和卡片都来自 uview-plus，当前项目已经可以直接按需使用 `up-` 组件。
			</text>
		</view>

		<up-card title="菜谱搜索" sub-title="uview-plus demo" :show-foot="false">
			<template #body>
				<view class="panel">
					<up-input
						v-model="keyword"
						placeholder="输入想吃的菜，比如番茄炒蛋"
						clearable
						prefixIcon="search"
						border="surround"
					></up-input>
					<view class="panel__actions">
						<up-button
							type="primary"
							text="添加偏好"
							shape="circle"
							@click="addPreference"
						></up-button>
						<up-button
							type="success"
							text="填充示例"
							plain
							shape="circle"
							@click="fillExample"
						></up-button>
					</view>
				</view>
			</template>
		</up-card>

		<up-card title="当前偏好" sub-title="本地示例" :show-foot="false">
			<template #body>
				<view v-if="preferences.length" class="list">
					<view v-for="item in preferences" :key="item" class="list__item">
						<text class="list__name">{{ item }}</text>
						<up-button
							type="error"
							text="删除"
							size="mini"
							plain
							@click="removePreference(item)"
						></up-button>
					</view>
				</view>
				<view v-else class="empty">
					<text class="empty__text">还没有记录任何口味偏好</text>
				</view>
			</template>
		</up-card>
	</view>
</template>

<script>
	export default {
		data() {
			return {
				keyword: '',
				noticeText: 'uview-plus 已接入，当前页面使用的是 up-notice-bar、up-card、up-input、up-button。',
				preferences: ['少油', '低糖', '高蛋白']
			}
		},
		methods: {
			addPreference() {
				const value = this.keyword.trim()
				if (!value) {
					uni.showToast({
						title: '先输入一个菜名或偏好',
						icon: 'none'
					})
					return
				}
				if (!this.preferences.includes(value)) {
					this.preferences.unshift(value)
				}
				this.noticeText = `最近添加：${value}`
				this.keyword = ''
			},
			fillExample() {
				this.keyword = '番茄炒蛋'
			},
			removePreference(item) {
				this.preferences = this.preferences.filter((value) => value !== item)
			}
		}
	}
</script>

<style lang="scss" scoped>
	.page {
		padding: 24rpx;
		display: flex;
		flex-direction: column;
		gap: 24rpx;
	}

	.hero {
		padding: 12rpx 8rpx 4rpx;
		display: flex;
		flex-direction: column;
		gap: 16rpx;
	}

	.hero__eyebrow {
		font-size: 24rpx;
		font-weight: 600;
		color: #3c9cff;
		letter-spacing: 2rpx;
	}

	.hero__title {
		font-size: 48rpx;
		line-height: 1.2;
		font-weight: 700;
		color: #1f2937;
	}

	.hero__desc {
		font-size: 28rpx;
		line-height: 1.7;
		color: #667085;
	}

	.panel {
		display: flex;
		flex-direction: column;
		gap: 24rpx;
	}

	.panel__actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 20rpx;
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: 20rpx;
	}

	.list__item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20rpx;
		padding: 20rpx 0;
		border-bottom: 1px solid #eef2f7;
	}

	.list__item:last-child {
		border-bottom: 0;
		padding-bottom: 0;
	}

	.list__name {
		font-size: 28rpx;
		color: #344054;
	}

	.empty {
		padding: 12rpx 0;
	}

	.empty__text {
		font-size: 28rpx;
		color: #98a2b3;
	}
</style>
